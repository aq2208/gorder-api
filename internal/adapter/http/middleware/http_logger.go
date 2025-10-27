package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/aq2208/gorder-api/internal/logging"
	"github.com/gin-gonic/gin"
)

const (
	reqBodyLimit  = 8 * 1024 // 8KB
	respBodyLimit = 8 * 1024 // 8KB
)

type bodyLogWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	// copy into buffer with cap
	if w.buf != nil && w.buf.Len() < respBodyLimit {
		remain := respBodyLimit - w.buf.Len()
		if len(b) > remain {
			w.buf.Write(b[:remain])
		} else {
			w.buf.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

func redactJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var m any
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw // not JSON
	}
	var scrub func(any) any
	scrub = func(x any) any {
		switch v := x.(type) {
		case map[string]any:
			for k, val := range v {
				kl := strings.ToLower(k)
				if kl == "password" || kl == "authorization" || kl == "token" || kl == "secret" {
					v[k] = "***redacted***"
					continue
				}
				v[k] = scrub(val)
			}
			return v
		case []any:
			for i := range v {
				v[i] = scrub(v[i])
			}
			return v
		default:
			return v
		}
	}
	out := scrub(m)
	b, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return b
}

func readCapped(rc io.ReadCloser, n int) (body []byte, truncated bool) {
	defer rc.Close()
	var buf bytes.Buffer
	_, _ = io.CopyN(&buf, rc, int64(n+1)) // read up to n+1
	b := buf.Bytes()
	if len(b) > n {
		return b[:n], true
	}
	return b, false
}

// Logging returns a Gin middleware that logs request/response and injects a slog.Logger into the context.
func Logging(base *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// request id
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = time.Now().UTC().Format("20060102T150405.000000000")
			c.Request.Header.Set("X-Request-Id", reqID)
		}
		c.Header("X-Request-Id", reqID)

		l := base.With(
			"req_id", reqID,
			"method", c.Request.Method,
			"path", c.FullPath(), // may be empty if no route matched
			"remote", c.ClientIP(),
		)
		logging.With(c, l)

		// capture request body (JSON only)
		var reqBodyLogged string
		ct := c.GetHeader("Content-Type")
		if strings.Contains(ct, "application/json") && c.Request.Body != nil {
			body, truncated := readCapped(c.Request.Body, reqBodyLimit)
			body = redactJSON(body)
			if truncated {
				body = append(body, []byte("...truncated...")...)
			}
			reqBodyLogged = string(body)
			// restore body for next handlers
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}

		// capture response
		blw := &bodyLogWriter{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = blw

		// process
		c.Next()

		status := c.Writer.Status()
		durMs := time.Since(start).Milliseconds()

		// response body only if JSON
		var respBodyLogged string
		if strings.Contains(c.Writer.Header().Get("Content-Type"), "application/json") {
			respBodyLogged = string(redactJSON(blw.buf.Bytes()))
			if blw.buf.Len() >= respBodyLimit {
				respBodyLogged += "...truncated..."
			}
		}

		attrs := []any{
			"status", status,
			"dur_ms", durMs,
		}
		if reqBodyLogged != "" {
			attrs = append(attrs, "req_body", reqBodyLogged)
		}
		if respBodyLogged != "" {
			attrs = append(attrs, "resp_body", respBodyLogged)
		}
		// include route params for convenience
		if len(c.Params) > 0 {
			attrs = append(attrs, "params", c.Params)
		}

		// include error (if any)
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.String())
		}

		// also log response size
		attrs = append(attrs, "resp_bytes", strconv.FormatInt(int64(c.Writer.Size()), 10))

		if status >= http.StatusBadRequest {
			l.Error("http_request", attrs...)
			return
		}
		l.Info("http_request", attrs...)
	}
}
