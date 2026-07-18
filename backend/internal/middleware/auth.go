package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int64   `json:"user_id"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	jwt.RegisteredClaims
}

// TokenValidator API Token 验证接口
type TokenValidator interface {
	ValidateToken(ctx context.Context, rawToken string) (int64, error)
}

// JWTAuth JWT + API Token 认证中间件
func JWTAuth(secret string, tokenValidator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证格式错误"})
			c.Abort()
			return
		}

		tokenStr := parts[1]

		// 尝试 API Token 验证（以 kada_ 开头）
		if strings.HasPrefix(tokenStr, "kada_") {
			if tokenValidator != nil {
				userID, err := tokenValidator.ValidateToken(c.Request.Context(), tokenStr)
				if err == nil {
					c.Set("user_id", userID)
					c.Next()
					return
				}
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的 API Token"})
			c.Abort()
			return
		}

		// JWT 验证
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证令牌无效或已过期"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证令牌解析失败"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_phone", claims.Phone)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}

// GetUserID 从 gin.Context 获取当前用户 ID
func GetUserID(c *gin.Context) int64 {
	id, _ := c.Get("user_id")
	return id.(int64)
}
