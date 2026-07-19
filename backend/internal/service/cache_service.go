package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/chun/kada-backend/internal/domain"
)

// CacheService Redis 缓存服务
type CacheService struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{
		client: client,
		ttl:    10 * time.Minute,
	}
}

// key 生成缓存 key
func (cs *CacheService) key(prefix, identifier string) string {
	return fmt.Sprintf("cache:%s:%s", prefix, identifier)
}

// GetLink 从缓存获取链接信息（按短码）
func (cs *CacheService) GetLink(ctx context.Context, shortCode string) (*domain.LinkInfo, bool) {
	data, err := cs.client.Get(ctx, cs.key("link", shortCode)).Bytes()
	if err != nil {
		return nil, false
	}

	var info domain.LinkInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, false
	}
	return &info, true
}

// SetLink 缓存链接信息
func (cs *CacheService) SetLink(ctx context.Context, info *domain.LinkInfo) {
	key := cs.key("link", info.ShortCode)
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	cs.client.Set(ctx, key, data, cs.ttl)
}

// InvalidateLink 使链接缓存失效
func (cs *CacheService) InvalidateLink(ctx context.Context, shortCode string) {
	cs.client.Del(ctx, cs.key("link", shortCode))
}
