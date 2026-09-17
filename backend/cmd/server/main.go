package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/chun/kada-backend/config"
	aiHandler "github.com/chun/kada-backend/internal/handler/ai"
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
	// Load the .env file
	_ = godotenv.Load()

	// Load configuration
	cfg := config.Load()

	// Security: release mode forbids the default/weak JWT secret, otherwise anyone could forge login tokens
	if os.Getenv("GIN_MODE") == "release" && config.IsWeakJWTSecret(cfg.JWTSecret) {
		log.Fatal("❌ refusing to use the default JWT_SECRET in production; set a strong random secret (e.g. openssl rand -hex 32)")
	}

	// Connect to the database
	db, err := infra.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer infra.CloseDB(db)

	// Apply the schema (GORM AutoMigrate owns the schema now; see internal/domain/entity).
	// In production DB_AUTO_MIGRATE is normally false and the schema is applied by cmd/migrate, so a
	// plain restart can never change the database.
	if cfg.AutoMigrate {
		if migrateErr := infra.Migrate(db); migrateErr != nil {
			log.Fatalf("Failed to migrate database: %v", migrateErr)
		}
	} else {
		log.Println("DB_AUTO_MIGRATE=false: skipping schema migration (run cmd/migrate to apply it)")
	}

	// Connect to Redis: keep the client even on initial ping failure (go-redis reconnects automatically).
	// Rate limiting fails open until Redis recovers, then works again with no process restart.
	redisClient, err := infra.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Printf("⚠️  Redis temporarily unavailable (rate limiting will fail open until it recovers): %v", err)
	}
	if redisClient != nil {
		defer infra.CloseRedis(redisClient)
	}

	// Initialize the Aliyun SMS verification service
	var smsSender service.SMSSender
	if cfg.SMSCredentialsConfigured() {
		smsSender, err = sms.NewAliyunSender(cfg.SMSAccessKeyID, cfg.SMSAccessKeySecret, cfg.SMSSignName, cfg.SMSTemplateCode)
		if err != nil {
			log.Printf("⚠️  failed to initialize SMS service: %v", err)
		}
	} else {
		// Distinguish "not configured" from "configured with the .env.example placeholders": both end in the
		// same warning, but the second one is the common cause of sign-up failing with a provider error.
		if cfg.SMSAccessKeyID != "" || cfg.SMSAccessKeySecret != "" {
			log.Println("⚠️  SMS_ACCESS_KEY_ID/SMS_ACCESS_KEY_SECRET still hold the .env.example placeholder values; " +
				"SMS is disabled and phone sign-up cannot work until real Aliyun credentials are set")
		} else {
			log.Println("⚠️  SMS service not configured; verification codes will only be printed to the log")
		}
	}

	// Initialize the cache service (if Redis is available)
	var cacheSvc *service.CacheService
	if redisClient != nil {
		cacheSvc = service.NewCacheService(redisClient)
	}

	// Kafka click event publisher (returns nil when there is no broker = disabled)
	kafkaPub := mq.NewKafkaClickPublisher(cfg.Brokers(), cfg.KafkaTopic)
	if kafkaPub != nil {
		defer kafkaPub.Close()
	}

	// Initialize the service layer
	authSvc := service.NewAuthService(db, cfg.JWTSecret, cfg.JWTExpires, smsSender)
	linkSvc := service.NewLinkService(db, cfg.BaseURL, cacheSvc, kafkaPub, service.NewClickStore(db))
	domainSvc := service.NewDomainService(db)
	folderSvc := service.NewFolderService(db)
	tagSvc := service.NewTagService(db)
	utmSvc := service.NewUTMTemplateService(db)
	tokenSvc := service.NewAPITokenService(db)
	workspaceSvc := service.NewWorkspaceService(db)

	// Initialize the handler layer
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

	// AI gateway: proxies /api/ai/* to the internal Python AI service
	aiH, err := aiHandler.NewHandler(cfg.AIBaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize AI gateway: %v", err)
	}

	// JWT + API Token middleware
	authMW := middleware.JWTAuth(cfg.JWTSecret, tokenSvc)

	// Rate limiting middleware
	var rateLimiter *middleware.RateLimiter
	if redisClient != nil {
		rateLimiter = middleware.NewRateLimiter(redisClient)
	}

	// Create the Gin engine
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Security: trust no X-Forwarded-For sent by any proxy (clients can forge it; it has been used to bypass rate limiting).
	// The real client IP comes from the X-Real-IP header rewritten by nginx (see middleware.RealIP).
	_ = r.SetTrustedProxies(nil)

	// Global rate limiting (if Redis is available)
	if rateLimiter != nil {
		r.Use(rateLimiter.Normal())
	}

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "kada-api",
			"version": "0.2.0",
		})
	})

	// Short-link redirection (public endpoint, high-traffic rate limit)
	var redirectMW gin.HandlerFunc
	if rateLimiter != nil {
		redirectMW = rateLimiter.Redirect()
	}
	redirectH.RegisterRoutes(r, redirectMW)

	// API v1 route group
	v1 := r.Group("/api")
	{
		// Auth routes (strict rate limit)
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
		aiH.RegisterRoutes(v1, authMW)
	}

	// Start the server (http.Server for graceful shutdown)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second, // gosec G112: prevent Slowloris slow-header attacks
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Printf("🚀 Kada API server starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown: wait for SIGINT/SIGTERM and give in-flight requests up to 10 seconds to finish
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	log.Println("🔻 shutdown signal received, starting graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}
