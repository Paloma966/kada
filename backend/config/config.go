package config

import (
	"os"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	JWTExpires  string
	BaseURL     string
	FrontendURL string

	// SMS service
	SMSAccessKeyID     string
	SMSAccessKeySecret string
	SMSSignName        string
	SMSTemplateCode    string

	// WeChat
	WechatAppID     string
	WechatAppSecret string

	// Kafka (click event stream; empty = disabled)
	KafkaBrokers string
	KafkaTopic   string
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
		KafkaBrokers:       getEnv("KAFKA_BROKERS", ""),
		KafkaTopic:         getEnv("KAFKA_TOPIC", "clicks"),
	}
}

// Brokers splits the comma-separated broker list, trimming whitespace and empty entries
func (c *Config) Brokers() []string {
	return SplitBrokers(c.KafkaBrokers)
}

// SplitBrokers splits the comma-separated broker list, trimming whitespace and empty entries (shared by server and worker)
func SplitBrokers(raw string) []string {
	var out []string
	for _, b := range strings.Split(raw, ",") {
		if b = strings.TrimSpace(b); b != "" {
			out = append(out, b)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// weakJWTSecrets lists known weak default secrets: startup fails outright in release mode
var weakJWTSecrets = []string{
	"",
	"kada-dev-secret-change-in-production",
	"changeme",
	"secret",
	"jwt-secret",
}

// IsWeakJWTSecret reports whether the secret is a known weak one
func IsWeakJWTSecret(s string) bool {
	for _, w := range weakJWTSecrets {
		if s == w {
			return true
		}
	}
	return false
}
