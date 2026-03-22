package middleware

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"k8s-wizard/api/models"

	"github.com/gin-gonic/gin"
)

const (
	contextKeyAuthRole = "auth.role"
	roleAnonymous      = "anonymous"
	roleUser           = "user"
	roleAdmin          = "admin"
)

var dangerousVerbPattern = regexp.MustCompile(`(?i)\b(create|delete|scale)\b|创建|删除|扩容|缩容|伸缩`)

// AuthConfig holds auth-related runtime configuration.
type AuthConfig struct {
	RequireAuth bool
	UserToken   string
	AdminToken  string
}

// NewAuthConfigFromEnv creates auth configuration from environment variables.
//
// Environment variables:
// - K8S_WIZARD_ENV: set to "production"/"prod" to default auth to required.
// - K8S_WIZARD_AUTH_REQUIRED: explicit override (true/false).
// - K8S_WIZARD_API_TOKEN: bearer token for authenticated API access.
// - K8S_WIZARD_ADMIN_TOKEN: admin token; defaults to API token when omitted.
func NewAuthConfigFromEnv() AuthConfig {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("K8S_WIZARD_ENV")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	}
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))
	}

	requireAuth := env == "production" || env == "prod"
	if v, ok := parseBoolEnv("K8S_WIZARD_AUTH_REQUIRED"); ok {
		requireAuth = v
	}

	userToken := strings.TrimSpace(os.Getenv("K8S_WIZARD_API_TOKEN"))
	adminToken := strings.TrimSpace(os.Getenv("K8S_WIZARD_ADMIN_TOKEN"))
	if adminToken == "" {
		adminToken = userToken
	}

	return AuthConfig{
		RequireAuth: requireAuth,
		UserToken:   userToken,
		AdminToken:  adminToken,
	}
}

func parseBoolEnv(name string) (bool, bool) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return false, false
	}

	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

// TokenAuth validates bearer tokens and sets request role in context.
func TokenAuth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		headerToken, hasToken := bearerToken(c.GetHeader("Authorization"))

		if !hasToken {
			if cfg.RequireAuth {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
				return
			}
			c.Set(contextKeyAuthRole, roleAnonymous)
			c.Next()
			return
		}

		if tokenEquals(headerToken, cfg.AdminToken) {
			c.Set(contextKeyAuthRole, roleAdmin)
			c.Next()
			return
		}
		if tokenEquals(headerToken, cfg.UserToken) {
			c.Set(contextKeyAuthRole, roleUser)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authentication token"})
	}
}

func bearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}
	const prefix = "bearer "
	if len(header) < len(prefix) || strings.ToLower(header[:len(prefix)]) != prefix {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

func tokenEquals(got, expected string) bool {
	if expected == "" || got == "" {
		return false
	}
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// RequireDangerousOperationAuth restricts dangerous chat operations to admin role.
func RequireDangerousOperationAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isDangerousChatRequest(c) {
			c.Next()
			return
		}

		role := roleAnonymous
		if v, exists := c.Get(contextKeyAuthRole); exists {
			if s, ok := v.(string); ok {
				role = s
			}
		}

		if role != roleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "dangerous operations require admin authorization"})
			return
		}

		c.Next()
	}
}

func isDangerousChatRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(bytes.TrimSpace(body)) == 0 {
		return false
	}

	var req models.ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}

	if dangerousVerbPattern.MatchString(req.Content) {
		return true
	}

	for k, v := range req.FormData {
		kl := strings.ToLower(strings.TrimSpace(k))
		if kl != "action" && kl != "type" && kl != "operation" && kl != "method" && kl != "tool" && kl != "tool_name" {
			continue
		}
		if dangerousVerbPattern.MatchString(strings.TrimSpace(toString(v))) {
			return true
		}
	}

	return false
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
