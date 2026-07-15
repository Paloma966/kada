package config

import (
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	JWTExpires  string
	BaseURL     string
	FrontendURL string

	// 短信服务
	SMSAccessKeyID     string
	SMSAccessKeySecret string
	SMSSignName        string
	SMSTemplateCode    string

	// 微信
	WechatAppID     string
	WechatAppSecret string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://kada:kada123@localhost:5432/kada?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:          getEnv("JWT_SECRET", "kada-dev-secret-change-in-production"),
		JWTExpires:         getEnv("JWT_EXPIRES_IN", "720h"),
		BaseURL:            getEnv("API_BASE_URL", "https://kada.click"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		SMSAccessKeyID:     getEnv("SMS_ACCESS_KEY_ID", ""),
		SMSAccessKeySecret: getEnv("SMS_ACCESS_KEY_SECRET", ""),
		SMSSignName:        getEnv("SMS_SIGN_NAME", "kada"),
		SMSTemplateCode:    getEnv("SMS_TEMPLATE_CODE", ""),
		WechatAppID:        getEnv("WECHAT_APP_ID", ""),
		WechatAppSecret:    getEnv("WECHAT_APP_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
