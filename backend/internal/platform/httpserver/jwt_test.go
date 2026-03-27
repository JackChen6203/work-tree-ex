package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	pjwt "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/jwt"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/response"
)

func setupJWTTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("JWT_SECRET", "test-secret-for-jwt")
	r := gin.New()
	r.Use(jwtMiddleware())
	r.GET("/api/v1/auth/session", func(c *gin.Context) {
		response.JSON(c, http.StatusOK, gin.H{"public": true})
	})
	r.GET("/healthz", func(c *gin.Context) {
		response.JSON(c, http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		response.JSON(c, http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/api/v1/trips", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		response.JSON(c, http.StatusOK, gin.H{"userID": userID})
	})
	return r
}

func TestJWTMiddlewareAllowsPublicRoutes(t *testing.T) {
	r := setupJWTTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for healthz, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for auth route, got %d", w2.Code)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200 for readyz, got %d", w3.Code)
	}
}

func TestJWTMiddlewareRejectsWithoutToken(t *testing.T) {
	r := setupJWTTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestJWTMiddlewareAcceptsValidToken(t *testing.T) {
	r := setupJWTTestEngine(t)

	token, err := pjwt.Generate("user-123", "test@example.com", 15*time.Minute)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJWTMiddlewareRejectsInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Middleware reads secret at request time
	r.Use(jwtMiddleware())
	r.GET("/api/v1/trips", func(c *gin.Context) {
		response.JSON(c, http.StatusOK, gin.H{"ok": true})
	})

	// Generate with one secret
	t.Setenv("JWT_SECRET", "secret-A")
	token, _ := pjwt.Generate("user-123", "test@example.com", 15*time.Minute)

	// Validate with different secret
	t.Setenv("JWT_SECRET", "secret-B")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad signature, got %d", w.Code)
	}
}

func TestJWTMiddlewareRejectsMalformedBearer(t *testing.T) {
	r := setupJWTTestEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for non-Bearer, got %d", w.Code)
	}
}
