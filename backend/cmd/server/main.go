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
	"github.com/chun/kada-backend/internal/mq"
	"github.com/chun/kada-backend/internal/service"
)

func main() {
	// 加载 .env 文件
	_ = godotenv.Load()

	// 加载配置
	cfg := config.Load()

	// 安全：release 模式禁止使用默认/弱 JWT 密钥，否则任何人均可伪造登录令牌
	if os.Getenv("GIN_MODE") == "release" && config.IsWeakJWTSecret(cfg.JWTSecret) {
		log.Fatal("❌ 生产环境禁止使用默认 JWT_SECRET，请设置强随机密钥（如 openssl rand -hex 32）")
	}

	// 连接数据库
	db, err := infra.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer infra.CloseDB(db)

	// 连接 Redis：初始 ping 失败也保留客户端（go-redis 自动重连）。
	// 限流在 Redis 恢复前 fail-open，恢复后自动生效，无需重启进程。
	redisClient, err := infra.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Printf("⚠️  Redis 暂时不可用（限流将暂时放行，恢复后自动生效）: %v", err)
	}
	if redisClient != nil {
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

	// Kafka 点击事件发布者（无 broker 时返回 nil = 禁用）
	kafkaPub := mq.NewKafkaClickPublisher(cfg.Brokers(), cfg.KafkaTopic)
	if kafkaPub != nil {
		defer kafkaPub.Close()
	}

	// 初始化 Service 层
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.JWTExpires, smsSender)
	linkSvc := service.NewLinkService(db, cfg.BaseURL, cacheSvc, kafkaPub, service.NewClickStore(db))
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

	// 安全：不信任任何代理传入的 X-Forwarded-For（客户端可伪造，曾被用于绕过限流）。
	// 真实客户端 IP 通过 nginx 覆写的 X-Real-IP 获取（见 middleware.RealIP）。
	_ = r.SetTrustedProxies(nil)

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
