package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RealIP 获取真实客户端 IP。
// 后端不信任任何客户端可伪造的代理头（X-Forwarded-For 可被伪造用于绕过限流）；
// 仅接受 nginx 覆写的 X-Real-IP（nginx 以 $remote_addr 设置，客户端无法伪造），
// 直连（无 nginx）时回退到 RemoteAddr。
func RealIP(c *gin.Context) string {
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		if parsed := net.ParseIP(strings.TrimSpace(ip)); parsed != nil {
			return parsed.String()
		}
	}
	return c.ClientIP()
}

// RateLimiter 基于 Redis 滑动窗口的速率限制中间件
type RateLimiter struct {
	client *redis.Client
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	Window  time.Duration               // 时间窗口
	Limit   int                         // 窗口内最大请求数
	KeyFunc func(c *gin.Context) string // 自定义 key 生成函数（默认使用 IP）
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Limit 返回 Gin 中间件
func (rl *RateLimiter) Limit(cfg RateLimitConfig) gin.HandlerFunc {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = defaultKeyFunc
	}

	return func(c *gin.Context) {
		key := "ratelimit:" + cfg.KeyFunc(c)
		now := time.Now().UnixNano()
		windowStart := now - cfg.Window.Nanoseconds()

		// 使用管道批量执行
		pipe := rl.client.Pipeline()
		// 移除窗口外的旧记录
		pipe.ZRemRangeByScore(c.Request.Context(), key,
			"0",
			fmt.Sprintf("%d", windowStart))
		// 统计当前窗口内的请求数
		countCmd := pipe.ZCard(c.Request.Context(), key)
		// 添加当前请求
		pipe.ZAdd(c.Request.Context(), key, redis.Z{
			Score:  float64(now),
			Member: fmt.Sprintf("%d", now),
		})
		// 设置 key 过期时间
		pipe.Expire(c.Request.Context(), key, cfg.Window*2)

		if _, err := pipe.Exec(c.Request.Context()); err != nil {
			// Redis 不可用时放行
			c.Next()
			return
		}

		count, _ := countCmd.Result()

		// 设置速率限制响应头
		remaining := cfg.Limit - int(count) - 1
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if int(count) >= cfg.Limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":               "请求过于频繁，请稍后再试",
				"retry_after_seconds": int(cfg.Window.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Strict 严格模式 - 认证接口（登录、注册、发送验证码）
func (rl *RateLimiter) Strict() gin.HandlerFunc {
	return rl.Limit(RateLimitConfig{
		Window: 1 * time.Minute,
		Limit:  20,
	})
}

// Normal 普通模式 - 通用 API
func (rl *RateLimiter) Normal() gin.HandlerFunc {
	return rl.Limit(RateLimitConfig{
		Window: 1 * time.Minute,
		Limit:  120,
	})
}

// Redirect 重定向模式 - 短链跳转（高频）
func (rl *RateLimiter) Redirect() gin.HandlerFunc {
	return rl.Limit(RateLimitConfig{
		Window: 1 * time.Minute,
		Limit:  300,
	})
}

func defaultKeyFunc(c *gin.Context) string {
	return RealIP(c)
}
