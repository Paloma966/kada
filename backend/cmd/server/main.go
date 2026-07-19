package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/chun/kada-backend/config"
	analyticsHandler "github.com/chun/kada-backend/internal/handler/analytics"
	authHandler "github.com/chun/kada-backend/internal/handler/auth"
	domainHandler "github.com/chun/kada-backend/internal/handler/domain"
	folderHandler "github.com/chun/kada-backend/internal/handler/folder"
	linkHandler "github.com/chun/kada-backend/internal/handler/link"
	redirectHandler "github.com/chun/kada-backend/internal/handler/redirect"
	tagHandler "github.com/chun/kada-backend/internal/handler/tag"
	tokenHandler "github.com/chun/kada-backend/internal/handler/token"
	utmHandler "github.com/chun/kada-backend/internal/handler/utm"
	workspaceHandler "github.com/chun/kada-backend/internal/handler/workspace"
	"github.com/chun/kada-backend/internal/infra"
	"github.com/chun/kada-backend/internal/infra/sms"
	"github.com/chun/kada-backend/internal/middleware"
	"github.com/chun/kada-backend/internal/service"
)

func main() {
	// 加载 .env 文件
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

	// 初始化阿里云短信认证服务
	var smsSender service.SMSSender
	if cfg.SMSAccessKeyID != "" && cfg.SMSAccessKeySecret != "" {
		smsSender, err = sms.NewAliyunSender(cfg.SMSAccessKeyID, cfg.SMSAccessKeySecret)
		if err != nil {
			log.Printf("⚠️  短信服务初始化失败: %v", err)
		}
	} else {
		log.Println("⚠️  未配置短信服务，验证码将只打印在日志中")
	}

	// 初始化缓存服务（如果 Redis 可用）
	var cacheSvc *service.CacheService
	if redisClient != nil {
		cacheSvc = service.NewCacheService(redisClient)
	}

	// 初始化 Service 层
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.JWTExpires, smsSender)
	linkSvc := service.NewLinkService(db, cfg.BaseURL, cacheSvc)
	domainSvc := service.NewDomainService(db)
	folderSvc := service.NewFolderService(db)
	tagSvc := service.NewTagService(db)
	utmSvc := service.NewUTMTemplateService(db)
	tokenSvc := service.NewAPITokenService(db)
	workspaceSvc := service.NewWorkspaceService(db)

	// 初始化 Handler 层
	authH := authHandler.NewHandler(authSvc)
	linkH := linkHandler.NewHandler(linkSvc)
	redirectH := redirectHandler.NewHandler(linkSvc)
	domainH := domainHandler.NewHandler(domainSvc)
	folderH := folderHandler.NewHandler(folderSvc)
	tagH := tagHandler.NewHandler(tagSvc)
	utmH := utmHandler.NewHandler(utmSvc)
	tokenH := tokenHandler.NewHandler(tokenSvc)
	workspaceH := workspaceHandler.NewHandler(workspaceSvc)
	analyticsH := analyticsHandler.NewHandler(db)

	// JWT + API Token 中间件
	authMW := middleware.JWTAuth(cfg.JWTSecret, tokenSvc)

	// 速率限制中间件
	var rateLimiter *middleware.RateLimiter
	if redisClient != nil {
		rateLimiter = middleware.NewRateLimiter(redisClient)
	}

	// 创建 Gin 实例
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// 全局速率限制（如果 Redis 可用）
	if rateLimiter != nil {
		r.Use(rateLimiter.Normal())
	}

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "kada-api",
			"version": "0.2.0",
		})
	})

	// 短链重定向（公开端点，高流量速率限制）
	var redirectMW gin.HandlerFunc
	if rateLimiter != nil {
		redirectMW = rateLimiter.Redirect()
	}
	redirectH.RegisterRoutes(r, redirectMW)

	// API v1 路由组
	v1 := r.Group("/api")
	{
		// 认证路由（严格速率限制）
		var strictMW gin.HandlerFunc
		if rateLimiter != nil {
			strictMW = rateLimiter.Strict()
		}
		authH.RegisterRoutes(v1, authMW, strictMW)
		linkH.RegisterRoutes(v1, authMW)
		domainH.RegisterRoutes(v1, authMW)
		folderH.RegisterRoutes(v1, authMW)
		tagH.RegisterRoutes(v1, authMW)
		utmH.RegisterRoutes(v1, authMW)
		tokenH.RegisterRoutes(v1, authMW)
		workspaceH.RegisterRoutes(v1, authMW)
		analyticsH.RegisterRoutes(v1, authMW)
	}

	// 启动服务器
	log.Printf("🚀 Kada API server starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
