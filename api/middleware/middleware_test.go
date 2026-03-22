package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTokenAuth_RequireAuthWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestTokenAuth_ValidUserToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.GET("/protected", func(c *gin.Context) {
		role, _ := c.Get(contextKeyAuthRole)
		c.JSON(http.StatusOK, gin.H{"role": role})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != "{\"role\":\"user\"}" {
		t.Fatalf("body = %s, want %s", body, "{\"role\":\"user\"}")
	}
}

func TestRequireDangerousOperationAuth_ForbidsNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.Use(RequireDangerousOperationAuth())
	router.POST("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"content":"delete deployment nginx"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireDangerousOperationAuth_AllowsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.Use(RequireDangerousOperationAuth())
	router.POST("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"content":"delete deployment nginx"}`))
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireDangerousOperationAuth_ForbidsNonAdminApply(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.Use(RequireDangerousOperationAuth())
	router.POST("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"content":"apply deployment nginx"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireDangerousOperationAuth_ForbidsNonAdminRestart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.Use(RequireDangerousOperationAuth())
	router.POST("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"content":"restart pod nginx"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireDangerousOperationAuth_ForbidsNonAdminExec(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.Use(RequireDangerousOperationAuth())
	router.POST("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"content":"exec into pod nginx"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireDangerousOperationAuth_RejectsLargeBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TokenAuth(AuthConfig{RequireAuth: true, UserToken: "user-token", AdminToken: "admin-token"}))
	router.Use(RequireDangerousOperationAuth())
	router.POST("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	oversized := strings.Repeat("a", (1<<20)+8)
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"content":"`+oversized+`"}`))
	req.Header.Set("Authorization", "Bearer user-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestCORS_AllowsConfiguredOriginAndHandlesPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orig := os.Getenv(allowedOriginsEnv)
	if err := os.Setenv(allowedOriginsEnv, "https://ui.example.com"); err != nil {
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
	router.OPTIONS("/chat", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/chat", nil)
	req.Header.Set("Origin", "https://ui.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://ui.example.com" {
		t.Fatalf("allow-origin = %q, want %q", got, "https://ui.example.com")
	}
}

func TestCORS_RejectsDisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orig := os.Getenv(allowedOriginsEnv)
	if err := os.Setenv(allowedOriginsEnv, "https://ui.example.com"); err != nil {
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
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}
