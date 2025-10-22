package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/aq2208/gorder-api/configs"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Authz struct {
	cfg configs.Config
}

func NewAuthz(cfg configs.Config) *Authz {
	return &Authz{cfg: cfg}
}

// Require checks JWT and ensures all required permissions are present
func (a *Authz) Require(requiredPerms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			unauth(c, "invalid_request", "missing bearer token")
			return
		}

		raw := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(a.cfg.Security.JWTSecret), nil
		}, jwt.WithLeeway(30*time.Second)) // small clock skew

		if err != nil || !token.Valid {
			unauth(c, "invalid_token", "invalid jwt")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			unauth(c, "invalid_token", "claims parsing error")
			return
		}

		if claims["iss"] != a.cfg.Security.Issuer || claims["aud"] != a.cfg.Security.Audience {
			unauth(c, "invalid_token", "iss/aud mismatch")
			return
		}

		perms := extractPerms(claims)
		if !hasAll(perms, requiredPerms) {
			forbidden(c, "insufficient_scope", "missing required permissions")
			return
		}

		c.Next()
	}
}

func extractPerms(claims jwt.MapClaims) map[string]string {
	out := map[string]string{}
	if arr, ok := claims["perms"].([]any); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok && s != "" {
				out[s] = ""
			}
		}
	}
	return out
}

func hasAll(have map[string]string, req []string) bool {
	for _, r := range req {
		if _, ok := have[r]; !ok {
			return false
		}
	}
	return true
}

func unauth(c *gin.Context, code, desc string) {
	c.Header("WWW-Authenticate", `Bearer error="`+code+`", error_description="`+desc+`"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": code, "error_description": desc})
}

func forbidden(c *gin.Context, code, desc string) {
	c.Header("WWW-Authenticate", `Bearer error="`+code+`", error_description="`+desc+`"`)
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": code, "error_description": desc})
}
