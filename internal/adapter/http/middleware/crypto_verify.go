package middleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/aq2208/gorder-api/internal/security"
	"github.com/gin-gonic/gin"
)

type CryptoVerify struct {
	cs security.CryptoService
}

func NewCryptoVerify(cs security.CryptoService) *CryptoVerify {
	return &CryptoVerify{cs: cs}
}

type EncryptedRequest struct {
	Data      string `json:"data"`      // base64 encoded ciphertext (nonce||ct)
	Signature string `json:"signature"` // base64 encoded RSA signature of ciphertext
}

type TextRequest struct {
	Text string `json:"text"`
}

func (cv *CryptoVerify) CryptoVerify() gin.HandlerFunc {
	return func(c *gin.Context) {
		// --- Read raw body ---
		rawBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}
		defer c.Request.Body.Close()

		// --- Parse outer wrapper ---
		var encReq EncryptedRequest
		if err := json.Unmarshal(rawBody, &encReq); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid encrypted request format"})
			return
		}

		// --- Decode Base64 fields ---
		ciphertext, err := base64.StdEncoding.DecodeString(encReq.Data)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid ciphertext encoding"})
			return
		}
		sig, err := base64.StdEncoding.DecodeString(encReq.Signature)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid signature encoding"})
			return
		}

		// --- Verify RSA-SHA256 signature ---
		if err := cv.cs.Verify(ciphertext, sig); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "signature verification failed"})
			return
		}

		// --- Decrypt AES256-GCM ciphertext ---
		plaintext, err := cv.cs.Decrypt(ciphertext)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "decryption failed"})
			return
		}

		// --- Replace the request body with decrypted plaintext ---
		c.Request.Body = io.NopCloser(bytes.NewReader(plaintext))
		c.Request.ContentLength = int64(len(plaintext))
		c.Request.Header.Set("Content-Type", "application/json")

		// Continue to the next handler
		c.Next()
	}
}

// EncryptAndSign body: raw JSON (any shape) - returns: { data: Base64(nonce||ciphertext), signature: Base64(RSA-SHA256(ciphertext)) }
func (cv *CryptoVerify) EncryptAndSign() gin.HandlerFunc {
	return func(c *gin.Context) {
		plaintext, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
			return
		}

		ct, err := cv.cs.Encrypt(plaintext)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt failed", "detail": err.Error()})
			return
		}

		sig, err := cv.cs.Sign(ct)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed (need RSA private key)", "detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":      base64.StdEncoding.EncodeToString(ct),
			"signature": base64.StdEncoding.EncodeToString(sig),
		})
	}
}

// EncryptAndSignText body: { "text": "..." } - returns: { data, signature }
func (cv *CryptoVerify) EncryptAndSignText() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TextRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON", "detail": err.Error()})
			return
		}

		plaintext := []byte(req.Text)

		ct, err := cv.cs.Encrypt(plaintext)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt failed", "detail": err.Error()})
			return
		}

		sig, err := cv.cs.Sign(ct)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sign failed (need RSA private key)", "detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":      base64.StdEncoding.EncodeToString(ct),
			"signature": base64.StdEncoding.EncodeToString(sig),
		})
	}
}
