package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const allowedOriginsEnv = "K8S_WIZARD_ALLOWED_ORIGINS"

var defaultAllowedOrigins = []string{
	"http://localhost:3000",
	"http://127.0.0.1:3000",
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

// CORSConfig holds runtime CORS configuration.
type CORSConfig struct {
	AllowAll       bool
	AllowOrigins   map[string]struct{}
	AllowCredsMode bool
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	cfg := newCORSConfigFromEnv()

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))

		if origin != "" {
			allowed, matchedOrigin := isOriginAllowed(origin, cfg)
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
				return
			}
			c.Writer.Header().Set("Access-Control-Allow-Origin", matchedOrigin)
		} else if cfg.AllowAll {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Vary", "Origin")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if cfg.AllowCredsMode {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func newCORSConfigFromEnv() CORSConfig {
	raw := strings.TrimSpace(os.Getenv(allowedOriginsEnv))
	origins := make(map[string]struct{})

	// Safe default for local development when env var is not configured.
	if raw == "" {
		for _, origin := range defaultAllowedOrigins {
			origins[origin] = struct{}{}
		}
		return CORSConfig{
			AllowAll:       false,
			AllowOrigins:   origins,
			AllowCredsMode: true,
		}
	}

	allowAll := false
	for _, token := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(token)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}
		origins[origin] = struct{}{}
	}

	return CORSConfig{
		AllowAll:       allowAll,
		AllowOrigins:   origins,
		AllowCredsMode: !allowAll,
	}
}

func isOriginAllowed(origin string, cfg CORSConfig) (bool, string) {
	if _, ok := cfg.AllowOrigins[origin]; ok {
		return true, origin
	}
	if cfg.AllowAll {
		return true, "*"
	}
	return false, ""
}
