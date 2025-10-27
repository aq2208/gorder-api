package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ctxKey struct{}

var (
	once sync.Once
	base *slog.Logger
)

// Init configures the global logger exactly once.
// Call this in main(): logging.Init("order-api", "./logs/app.log")
func Init(component, filePath string) *slog.Logger {
	once.Do(func() {
		_ = os.MkdirAll("../../logs", 0755)

		rot := &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    50, // MB
			MaxBackups: 3,
			MaxAge:     7, // days
			Compress:   false,
		}
		mw := io.MultiWriter(os.Stdout, rot)

		h := slog.NewJSONHandler(mw, &slog.HandlerOptions{Level: slog.LevelInfo})
		base = slog.New(h).With("component", component)
	})
	return base
}

// Base returns the global logger (Init if not already called).
func Base() *slog.Logger {
	if base == nil {
		// Safe default: initialize to ./logs/app.log with generic component
		return Init("app", "./logs/app.log")
	}
	return base
}

// New returns a child logger derived from the global one.
// IMPORTANT: does NOT create a new handler/writer; it reuses the global handler.
func New(component string) *slog.Logger {
	return Base().With("component", component)
}

// WithCtx stores a logger in a standard context (useful outside Gin).
func WithCtx(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromCtx fetches a logger from ctx or falls back to the global one.
func FromCtx(ctx context.Context) *slog.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return Base()
}

// With stores the logger in gin.Context.
func With(c *gin.Context, l *slog.Logger) {
	c.Set("logger", l)
}

// From returns the request-scoped logger from gin.Context, or the global one.
func From(c *gin.Context) *slog.Logger {
	if v, ok := c.Get("logger"); ok {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return Base()
}
