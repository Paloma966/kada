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

func TestRealIP_PrefersXRealIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	_ = r.SetTrustedProxies(nil) // 与生产一致：不信任代理头
	var got string
	r.GET("/test", func(c *gin.Context) { got = RealIP(c) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4") // 伪造值
	req.Header.Set("X-Real-IP", "5.6.7.8")       // nginx 覆写值
	r.ServeHTTP(w, req)

	if got != "5.6.7.8" {
		t.Errorf("expected X-Real-IP '5.6.7.8', got %q", got)
	}
}

func TestRealIP_IgnoresSpoofedXFF(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	_ = r.SetTrustedProxies(nil) // 与生产一致：不信任代理头
	var got string
	r.GET("/test", func(c *gin.Context) { got = RealIP(c) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4") // 仅伪造 XFF
	r.ServeHTTP(w, req)

	if got != "203.0.113.9" {
		t.Errorf("expected RemoteAddr '203.0.113.9' (spoofed XFF ignored), got %q", got)
	}
}

func TestRealIP_InvalidXRealIPFallsBack(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	var got string
	r.GET("/test", func(c *gin.Context) { got = RealIP(c) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Real-IP", "not-an-ip")
	r.ServeHTTP(w, req)

	if got != "203.0.113.9" {
		t.Errorf("expected RemoteAddr fallback '203.0.113.9', got %q", got)
	}
}
