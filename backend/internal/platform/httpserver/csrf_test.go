package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCSRFRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(csrfMiddleware())
	r.POST("/api/v1/trips", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})
	return r
}

func TestCSRFMiddlewareRequiresTokenForCookieSession(t *testing.T) {
	r := setupCSRFRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-1"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected missing csrf token status 403, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "CSRF token is required") {
		t.Fatalf("expected csrf error response, got %s", w.Body.String())
	}
}

func TestCSRFMiddlewareAcceptsMatchingDoubleSubmitToken(t *testing.T) {
	r := setupCSRFRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-1"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "token-1"})
	req.Header.Set(csrfHeaderName, "token-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected matching csrf token status 201, got %d", w.Code)
	}
}

func TestCSRFMiddlewareRejectsMismatchedDoubleSubmitToken(t *testing.T) {
	r := setupCSRFRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-1"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "token-1"})
	req.Header.Set(csrfHeaderName, "token-2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected mismatched csrf token status 403, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), csrfInvalidTokenResponse) {
		t.Fatalf("expected invalid csrf response, got %s", w.Body.String())
	}
}

func TestCSRFMiddlewareBootstrapsMissingCSRFCookie(t *testing.T) {
	r := setupCSRFRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "session-1"})
	req.Header.Set(csrfHeaderName, "token-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected bootstrap csrf token status 201, got %d", w.Code)
	}

	cookie := findCookie(w.Result().Cookies(), csrfCookieName)
	if cookie == nil || cookie.Value != "token-1" {
		t.Fatalf("expected csrf cookie to be bootstrapped, got %#v", cookie)
	}
}

func TestCSRFMiddlewareSkipsBearerRequests(t *testing.T) {
	r := setupCSRFRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected bearer request status 201, got %d", w.Code)
	}
}

func TestCORSMiddlewareAllowsCSRFHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware([]string{"http://localhost:5173"}))
	r.POST("/api/v1/trips", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/trips", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status 204, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), csrfHeaderName) {
		t.Fatalf("expected CORS allow headers to include %s, got %s", csrfHeaderName, w.Header().Get("Access-Control-Allow-Headers"))
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
