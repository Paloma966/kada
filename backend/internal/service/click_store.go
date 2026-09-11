package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ClickWriter writes click logs directly (shared by the production degraded fallback and the worker consumer)
type ClickWriter interface {
	WriteClick(ctx context.Context, eventID string, linkID int64, ip, userAgent, platform, referer string, createdAt time.Time) error
}

// ClickStore is the pgx implementation of ClickWriter
type ClickStore struct {
	db *pgxpool.Pool
}

func NewClickStore(db *pgxpool.Pool) *ClickStore {
	return &ClickStore{db: db}
}

// WriteClick inside a transaction: insert the click log + increment the counter.
// deduplicates by event_id: when Kafka redelivers, or the degraded direct write races the worker on the same event,
// the second INSERT hits the unique conflict (RowsAffected=0) and click_count is not incremented again.
func (s *ClickStore) WriteClick(ctx context.Context, eventID string, linkID int64, ip, userAgent, platform, referer string, createdAt time.Time) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		INSERT INTO click_logs (link_id, ip, user_agent, platform, referer, created_at, event_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (event_id) DO NOTHING
	`, linkID, ip, userAgent, platform, referer, createdAt, eventID)
	if err != nil {
		return err
	}

	// duplicate event: the log was already written, skip the counter increment and only commit (staying idempotent)
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE links SET click_count = click_count + 1 WHERE id = $1
	`, linkID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
