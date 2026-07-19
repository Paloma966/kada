package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDefaultKeyFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.168.1.100:54321"

	key := defaultKeyFunc(c)
	if key != "192.168.1.100" {
		t.Errorf("expected client IP '192.168.1.100', got %q", key)
	}
}

func TestNewRateLimiter_NilClient(t *testing.T) {
	// 即使传入 nil client 也不应 panic
	rl := NewRateLimiter(nil)
	if rl == nil {
		t.Fatal("NewRateLimiter should not return nil")
	}
	if rl.client != nil {
		t.Error("client should be nil")
	}
}
