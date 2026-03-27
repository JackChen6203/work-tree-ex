package response

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDatabaseUnavailableWrites503(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)
	c.Set("requestID", "req-test")

	handled := DatabaseUnavailable(c, context.DeadlineExceeded)
	if !handled {
		t.Fatalf("expected database unavailable error to be handled")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}
	if got := w.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("expected Retry-After=1, got %q", got)
	}

	var body map[string]map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"]["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("expected SERVICE_UNAVAILABLE, got %v", body["error"]["code"])
	}
}

func TestDatabaseUnavailablePassesThroughNonDBErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	handled := DatabaseUnavailable(c, context.Canceled)
	if handled {
		t.Fatalf("did not expect non-pool error to be handled")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected untouched recorder status 200, got %d", w.Code)
	}
}
