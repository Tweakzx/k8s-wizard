package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTokenAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestCORS_WildcardOriginDisablesCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orig := os.Getenv(allowedOriginsEnv)
	if err := os.Setenv(allowedOriginsEnv, "*"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() {
		if orig == "" {
			_ = os.Unsetenv(allowedOriginsEnv)
		} else {
			_ = os.Setenv(allowedOriginsEnv, orig)
		}
	})

	router := gin.New()
	router.Use(CORS())
	router.GET("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	req.Header.Set("Origin", "https://any-origin.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("allow-origin = %q, want %q", got, "*")
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("allow-credentials = %q, want empty", got)
	}
}
