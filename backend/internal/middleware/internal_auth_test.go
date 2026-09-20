package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInternalAI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const secret = "internal-test-secret"

	tests := []struct {
		name       string
		remoteAddr string
		secretHdr  string
		userHdr    string
		wantCode   int
	}{
		{
			name:       "missing secret header",
			remoteAddr: "127.0.0.1:1234",
			secretHdr:  "",
			userHdr:    "5",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "wrong secret",
			remoteAddr: "127.0.0.1:1234",
			secretHdr:  "nope",
			userHdr:    "5",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "public peer rejected",
			remoteAddr: "192.0.2.1:1234",
			secretHdr:  secret,
			userHdr:    "5",
			wantCode:   http.StatusForbidden,
		},
		{
			name:       "missing user id",
			remoteAddr: "127.0.0.1:1234",
			secretHdr:  secret,
			userHdr:    "",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "non-numeric user id",
			remoteAddr: "127.0.0.1:1234",
			secretHdr:  secret,
			userHdr:    "demo-user",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "zero user id",
			remoteAddr: "127.0.0.1:1234",
			secretHdr:  secret,
			userHdr:    "0",
			wantCode:   http.StatusUnauthorized,
		},
		{
			name:       "valid loopback callback",
			remoteAddr: "127.0.0.1:1234",
			secretHdr:  secret,
			userHdr:    "5",
			wantCode:   http.StatusOK,
		},
		{
			name:       "valid private network callback",
			remoteAddr: "172.18.0.7:1234",
			secretHdr:  secret,
			userHdr:    "5",
			wantCode:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(InternalAI(secret))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"user_id": GetUserID(c)})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.secretHdr != "" {
				req.Header.Set("X-Internal-Secret", tt.secretHdr)
			}
			if tt.userHdr != "" {
				req.Header.Set("X-Kada-User-ID", tt.userHdr)
			}
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("expected %d, got %d (body: %s)", tt.wantCode, w.Code, w.Body.String())
			}
			if tt.wantCode == http.StatusOK && w.Body.String() != `{"user_id":5}` {
				t.Fatalf("expected user_id 5 injected into context, got %s", w.Body.String())
			}
		})
	}
}

func TestInternalAI_EmptySecretRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(InternalAI(""))
	r.GET("/test", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Kada-User-ID", "5")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with empty configured secret, got %d", w.Code)
	}
}
