package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ClickWriter 直写点击日志（生产降级回退与 worker 消费共用）
type ClickWriter interface {
	WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string, createdAt time.Time) error
}

// ClickStore pgx 实现的 ClickWriter
type ClickStore struct {
	db *pgxpool.Pool
}

func NewClickStore(db *pgxpool.Pool) *ClickStore {
	return &ClickStore{db: db}
}

// WriteClick 事务内：插入点击日志 + 累加计数
func (s *ClickStore) WriteClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string, createdAt time.Time) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO click_logs (link_id, ip, user_agent, platform, referer, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, linkID, ip, userAgent, platform, referer, createdAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE links SET click_count = click_count + 1 WHERE id = $1
	`, linkID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
