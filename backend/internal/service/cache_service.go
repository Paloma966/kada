package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/chun/kada-backend/internal/domain"
)

// CacheService is a Redis cache service
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

// key builds a cache key
func (cs *CacheService) key(prefix, identifier string) string {
	return fmt.Sprintf("cache:%s:%s", prefix, identifier)
}

// GetLink gets link info from the cache (by short code)
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

// SetLink caches link info
func (cs *CacheService) SetLink(ctx context.Context, info *domain.LinkInfo) {
	key := cs.key("link", info.ShortCode)
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	cs.client.Set(ctx, key, data, cs.ttl)
}

// InvalidateLink invalidates the cached link
func (cs *CacheService) InvalidateLink(ctx context.Context, shortCode string) {
	cs.client.Del(ctx, cs.key("link", shortCode))
}
