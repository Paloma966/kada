// Package ai proxies AI chat requests to the internal Python AI service.
//
// Kada's browser-facing API only ever talks to the Python service through this
// gateway: the Go process owns authentication (JWT middleware), injects the
// authenticated user id as X-Kada-User-ID, and streams the SSE response back
// verbatim. The Python service listens on 127.0.0.1 only and is never exposed
// to the public internet.
package ai

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/middleware"
)

// userIDCtxKey carries the authenticated user id from the gin handler into the
// reverse-proxy Director. The Director only sees a *http.Request, not the
// gin.Context, so we smuggle the value through the request context.
type userIDCtxKey struct{}

// internalAuthHeader is the header that tells the Python service this request
// came through the gateway rather than from some other local process - which is
// what makes the X-Kada-User-ID header sent alongside it trustworthy. The value
// it carries is the shared secret, not the header name itself.
const internalAuthHeader = "X-Internal-Secret"

// Handler proxies /api/ai/* to the internal Python AI service.
type Handler struct {
	proxy *httputil.ReverseProxy
}

// NewHandler builds the gateway that forwards /api/ai/* to the Python service
// at aiBaseURL (e.g. http://127.0.0.1:8000). internalSecret is injected as
// X-Internal-Secret; an empty secret disables the check on both sides.
func NewHandler(aiBaseURL, internalSecret string) (*Handler, error) {
	target, err := url.Parse(aiBaseURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	// SSE must reach the client token by token; -1 = flush after every write
	// instead of buffering. Without this, the streaming reply arrives in
	// chunks rather than word by word.
	proxy.FlushInterval = -1
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		// If the SSE stream already started, headers are written and we can no
		// longer change the status code; just log and stop the stream.
		if w.Header().Get("Content-Type") != "" {
			log.Printf("ai proxy stream error: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(gin.H{"error": "AI service unavailable: " + err.Error()})
	}

	h := &Handler{proxy: proxy}
	baseDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		baseDirector(req) // sets scheme/host/query from the target URL

		// Map the public path to the Python path by dropping this gateway's mount
		// prefix: /api/ai/chat -> /chat, /api/ai/conversations/current ->
		// /conversations/current. The AI service owns its own paths and knows
		// nothing about /api/ai, so the two surfaces can move independently.
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/ai")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}

		// Security: never trust a client-supplied X-Kada-User-ID. The gateway
		// always overwrites it with the authenticated user id from the JWT,
		// so a caller can never read or write another user's conversations.
		req.Header.Del("X-Kada-User-ID")
		if userID, ok := req.Context().Value(userIDCtxKey{}).(int64); ok && userID > 0 {
			req.Header.Set("X-Kada-User-ID", strconv.FormatInt(userID, 10))
		}

		// Same rule for the service secret. Deleting matters even when no secret
		// is configured: the AI service must never see a header the gateway did
		// not send, or a caller could smuggle one in and make a direct request
		// look like it came through the gateway.
		req.Header.Del(internalAuthHeader)
		if internalSecret != "" {
			req.Header.Set(internalAuthHeader, internalSecret)
		}
	}
	return h, nil
}

// RegisterRoutes mounts the AI gateway on the /api router group.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	r.Use(authMW)
	r.POST("/ai/chat", h.forward)
	// Single-session mode: current returns the user's latest conversation with
	// its messages, restart drops it and creates a fresh empty one. The paths say
	// conversations, not "session": the table is ai_conversations and the
	// response already carries conversation_id.
	r.GET("/ai/conversations/current", h.forward)
	r.POST("/ai/conversations/restart", h.forward)
}

// forward proxies the request to the Python AI service.
func (h *Handler) forward(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user required"})
		return
	}

	// SSE is a long-lived connection: lift the server-level write timeout
	// (cmd/server sets 10s) for this request, otherwise Go would cut the
	// stream mid-answer. time.Time{} means "no deadline".
	if ctrl := http.NewResponseController(c.Writer); ctrl != nil {
		if err := ctrl.SetWriteDeadline(time.Time{}); err != nil {
			log.Printf("ai: could not clear write deadline: %v", err)
		}
	}

	req := c.Request.WithContext(context.WithValue(c.Request.Context(), userIDCtxKey{}, userID))
	h.proxy.ServeHTTP(c.Writer, req)
}
