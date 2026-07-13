package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/chun/kada-backend/config"
	authHandler "github.com/chun/kada-backend/internal/handler/auth"
	linkHandler "github.com/chun/kada-backend/internal/handler/link"
	redirectHandler "github.com/chun/kada-backend/internal/handler/redirect"
	"github.com/chun/kada-backend/internal/infra"
	"github.com/chun/kada-backend/internal/middleware"
	"github.com/chun/kada-backend/internal/service"
)

func main() {
	// 加载 .env 文件（如果存在）
	_ = godotenv.Load()

	// 加载配置
	cfg := config.Load()

	// 连接数据库
	db, err := infra.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer infra.CloseDB(db)

	// 连接 Redis
	redisClient, err := infra.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Printf("⚠️  Redis 连接失败（继续运行）: %v", err)
	} else {
		defer infra.CloseRedis(redisClient)
	}

	// 初始化 Service 层
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.JWTExpires)
	linkSvc := service.NewLinkService(db, cfg.BaseURL)

	// 初始化 Handler 层
	authH := authHandler.NewHandler(authSvc)
	linkH := linkHandler.NewHandler(linkSvc)
	redirectH := redirectHandler.NewHandler(linkSvc)

	// JWT 中间件
	authMW := middleware.JWTAuth(cfg.JWTSecret)

	// 创建 Gin 实例
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "kada-api",
			"version": "0.1.0",
		})
	})

	// 短链重定向（公开端点，无需认证）
	redirectH.RegisterRoutes(r)

	// API v1 路由组
	v1 := r.Group("/api")
	{
		authH.RegisterRoutes(v1, authMW)
		linkH.RegisterRoutes(v1, authMW)
	}

	// 启动服务器
	log.Printf("🚀 Kada API server starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
