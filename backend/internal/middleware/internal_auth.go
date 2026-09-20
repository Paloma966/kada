package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// InternalAI authenticates service-to-service callbacks from the Python AI
// service (POST /internal/ai/links, GET /internal/ai/analytics/overview).
//
// There is no end-user token here: the browser request already passed the
// JWT-protected /api/ai gateway, and the AI service forwards the authenticated
// user id in X-Kada-User-ID. Three independent gates keep these routes
// internal-only:
//  1. a shared secret (AI_INTERNAL_SECRET) known only to Go and the AI service,
//  2. the TCP peer must be loopback or a private address (host or compose network),
//  3. nginx has no public location mapping to /internal/.
//
// On success the user id is set under the same context key as JWTAuth, so the
// existing link/analytics handlers work unchanged.
func InternalAI(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" || c.GetHeader("X-Internal-Secret") != secret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid internal credentials"})
			c.Abort()
			return
		}

		// Use the real TCP peer, never X-Forwarded-For, which a client can forge.
		host := c.Request.RemoteAddr
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if !isPrivateHost(host) {
			c.JSON(http.StatusForbidden, gin.H{"error": "internal endpoint not reachable from this address"})
			c.Abort()
			return
		}

		userID, err := strconv.ParseInt(strings.TrimSpace(c.GetHeader("X-Kada-User-ID")), 10, 64)
		if err != nil || userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "valid X-Kada-User-ID required"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// isPrivateHost reports whether host is loopback, an RFC1918/ULA private
// address or a link-local address - peers that can only come from the host
// itself or the internal compose network.
func isPrivateHost(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
