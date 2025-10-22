package http

import (
	"net/http"
	"time"

	"github.com/aq2208/gorder-api/configs"
	"github.com/aq2208/gorder-api/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type TokenHandler struct {
	cfg configs.Config
}

func NewTokenHandler(cfg configs.Config) *TokenHandler {
	return &TokenHandler{cfg: cfg}
}

// POST /token (form or JSON)
// Accepts: client_id, client_secret
// Optional: scope (space-separated subset of client's perms)
func (h *TokenHandler) IssueToken(c *gin.Context) {
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	if clientID == "" || clientSecret == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid client"})
		return
	}

	cl, ok := security.Clients[clientID]
	if !ok || !cl.Enabled || clientSecret != cl.Secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid client"})
		return
	}

	perms := cl.Perms
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":      h.cfg.Security.Issuer,                            // issuer
		"aud":      h.cfg.Security.Audience,                          // audience
		"iat":      now.Unix(),                                       // issued at
		"nbf":      now.Unix(),                                       // not before
		"exp":      now.Add(h.cfg.Security.TTL * time.Minute).Unix(), // expire
		"clientID": clientID,
		"perms":    perms,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.cfg.Security.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": signed,
		"token_type":   "Bearer",
		"expires_in":   h.cfg.Security.TTL,
	})
}
