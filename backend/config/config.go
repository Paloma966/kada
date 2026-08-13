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

	// 短信服务
	SMSAccessKeyID     string
	SMSAccessKeySecret string
	SMSSignName        string
	SMSTemplateCode    string

	// 微信
	WechatAppID     string
	WechatAppSecret string

	// Kafka（点击事件流；空 = 禁用）
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

// Brokers 拆分逗号分隔的 broker 列表，去空白与空项
func (c *Config) Brokers() []string {
	return SplitBrokers(c.KafkaBrokers)
}

// SplitBrokers 拆分逗号分隔的 broker 列表，去空白与空项（server 与 worker 共用）
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
