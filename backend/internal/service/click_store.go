package service

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ClickWriter writes click logs directly (shared by the production degraded fallback and the worker consumer)
type ClickWriter interface {
	WriteClick(ctx context.Context, eventID string, linkID int64, ip, userAgent, platform, referer string, createdAt time.Time) error
}

// ClickStore is the GORM implementation of ClickWriter
type ClickStore struct {
	db *gorm.DB
}

func NewClickStore(db *gorm.DB) *ClickStore {
	return &ClickStore{db: db}
}

// WriteClick inside a transaction: insert the click log + increment the counter.
// deduplicates by event_id: when Kafka redelivers, or the degraded direct write races the worker on the same event,
// the second INSERT hits the unique conflict (RowsAffected=0) and click_count is not incremented again.
func (s *ClickStore) WriteClick(ctx context.Context, eventID string, linkID int64, ip, userAgent, platform, referer string, createdAt time.Time) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`
			INSERT INTO click_logs (link_id, ip, user_agent, platform, referer, created_at, event_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (event_id) DO NOTHING
		`, linkID, ip, userAgent, platform, referer, createdAt, eventID)
		if res.Error != nil {
			return res.Error
		}

		// duplicate event: the log was already written, skip the counter increment and only commit (staying idempotent)
		if res.RowsAffected == 0 {
			return nil
		}

		if err := tx.Exec(`UPDATE links SET click_count = click_count + 1 WHERE id = ?`, linkID).Error; err != nil {
			return err
		}
		return nil
	})
}
