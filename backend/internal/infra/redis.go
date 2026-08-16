package infra

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		// 初始 ping 失败也返回客户端：go-redis 内部自动重连。
		// 调用方（限流器）在 Redis 恢复前 fail-open，恢复后自动生效，无需重启进程。
		return client, fmt.Errorf("failed to ping Redis: %w", err)
	}

	log.Println("✅ Redis connected")
	return client, nil
}

func CloseRedis(client *redis.Client) {
	if err := client.Close(); err != nil {
		log.Printf("close redis failed: %v", err)
		return
	}
	log.Println("Redis connection closed")
}
