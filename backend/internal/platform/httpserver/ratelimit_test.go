package httpserver

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/cache"
)

func TestRateLimitKeyUsesUserAndPath(t *testing.T) {
	cache.SetDistributedMode(false)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/v1/trips/123/items", nil)
	c.Request = req
	c.Set("userID", "user-1")

	key := rateLimitKey(c)
	expected := "user:user-1|endpoint:/api/v1/trips/123/items"
	if key != expected {
		t.Fatalf("expected key %q, got %q", expected, key)
	}
}

func TestRateLimitKeyFallsBackToIP(t *testing.T) {
	cache.SetDistributedMode(false)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/api/v1/trips", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	c.Request = req

	key := rateLimitKey(c)
	expected := "ip:203.0.113.5"
	if key != expected {
		t.Fatalf("expected key %q, got %q", expected, key)
	}
}

func TestInMemoryRateLimiterBurstExhausted(t *testing.T) {
	cache.SetDistributedMode(false)
	limiter := newInMemoryRateLimiter(1, 2)
	key := "ip:203.0.113.5"

	allowed, remaining, err := limiter.allow(nil, key)
	if err != nil {
		t.Fatalf("allow #1 error: %v", err)
	}
	if !allowed || remaining != 1 {
		t.Fatalf("allow #1 expected allowed=true remaining=1, got allowed=%v remaining=%d", allowed, remaining)
	}

	allowed, remaining, err = limiter.allow(nil, key)
	if err != nil {
		t.Fatalf("allow #2 error: %v", err)
	}
	if !allowed || remaining != 0 {
		t.Fatalf("allow #2 expected allowed=true remaining=0, got allowed=%v remaining=%d", allowed, remaining)
	}

	allowed, remaining, err = limiter.allow(nil, key)
	if err != nil {
		t.Fatalf("allow #3 error: %v", err)
	}
	if allowed || remaining != 0 {
		t.Fatalf("allow #3 expected allowed=false remaining=0, got allowed=%v remaining=%d", allowed, remaining)
	}
}
