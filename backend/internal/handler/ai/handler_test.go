package ai

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeAuth mimics the real JWT middleware: it stamps a user id into the context.
func fakeAuth(userID int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

// notifyRecorder wraps httptest.ResponseRecorder with CloseNotify, which the
// reverse proxy probes to detect a disconnected client. The real net/http
// response implements it; httptest's recorder does not, so tests must.
type notifyRecorder struct {
	*httptest.ResponseRecorder
}

func (r *notifyRecorder) CloseNotify() <-chan bool { return nil }

// serve runs a request through the gin router and returns the wrapped recorder.
func serve(r *gin.Engine, method, target string, body *bytes.Buffer, headers map[string]string) *notifyRecorder {
	w := &notifyRecorder{ResponseRecorder: httptest.NewRecorder()}
	req := httptest.NewRequest(method, target, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	r.ServeHTTP(w, req)
	return w
}

func newTestRouter(t *testing.T, upstream string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h, err := NewHandler(upstream)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	r := gin.New()
	h.RegisterRoutes(r.Group("/api"), fakeAuth(42))
	return r
}

// The gateway must forward /api/ai/chat to /v1/chat, inject the authenticated
// user id, and stream the SSE body back verbatim.
func TestProxyForwardsChatWithUserID(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/chat" {
			t.Errorf("upstream path = %q, want /v1/chat", got)
		}
		if got := r.Header.Get("X-Kada-User-ID"); got != "42" {
			t.Errorf("X-Kada-User-ID = %q, want 42", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: token\ndata: {\"delta\":\"hi\"}\n\n"))
	}))
	defer upstream.Close()

	r := newTestRouter(t, upstream.URL)
	w := serve(r, http.MethodPost, "/api/ai/chat", bytes.NewBufferString(`{"message":"hi"}`), nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"delta":"hi"`) {
		t.Errorf("body = %q, want SSE token event", w.Body.String())
	}
}

// A client-supplied X-Kada-User-ID must never pass through: the gateway always
// overwrites it with the JWT user id, so users cannot impersonate others.
func TestProxyOverwritesClientUserID(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Kada-User-ID"); got != "42" {
			t.Errorf("X-Kada-User-ID = %q, want gateway-injected 42", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	r := newTestRouter(t, upstream.URL)
	// client tries to impersonate user 999
	w := serve(r, http.MethodPost, "/api/ai/chat", bytes.NewBufferString(`{}`),
		map[string]string{"X-Kada-User-ID": "999"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// Conversation paths must be remapped too: /api/ai/conversations/{id}/messages
// -> /v1/conversations/{id}/messages, keeping the id segment intact.
func TestProxyForwardsConversationPaths(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/conversations/abc-123/messages" {
			t.Errorf("upstream path = %q, want /v1/conversations/abc-123/messages", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[]}`))
	}))
	defer upstream.Close()

	r := newTestRouter(t, upstream.URL)
	w := serve(r, http.MethodGet, "/api/ai/conversations/abc-123/messages", bytes.NewBuffer(nil), nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}
