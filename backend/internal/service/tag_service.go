package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

type TagService struct {
	db *pgxpool.Pool
}

func NewTagService(db *pgxpool.Pool) *TagService {
	return &TagService{db: db}
}

func (s *TagService) Create(ctx context.Context, userID int64, req domain.CreateTagRequest) (*domain.Tag, error) {
	color := "#6366F1"
	if req.Color != nil && *req.Color != "" {
		color = *req.Color
	}
	var t domain.Tag
	err := s.db.QueryRow(ctx,
		`INSERT INTO tags (user_id, name, color) VALUES ($1, $2, $3) RETURNING id, user_id, name, color, created_at`,
		userID, req.Name, color,
	).Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt)
	if err != nil {
		return nil, errors.New("failed to create tag")
	}
	return &t, nil
}

func (s *TagService) List(ctx context.Context, userID int64) ([]domain.Tag, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, name, color, created_at FROM tags WHERE user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, errors.New("failed to list tags")
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, errors.New("failed to list tags")
		}
		tags = append(tags, t)
	}
	if tags == nil {
		tags = []domain.Tag{}
	}
	return tags, nil
}

func (s *TagService) Delete(ctx context.Context, userID, tagID int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM tags WHERE id = $1 AND user_id = $2`, tagID, userID)
	if err != nil {
		return errors.New("failed to delete tag")
	}
	return nil
}

// AddTagToLink adds a tag to a link
func (s *TagService) AddTagToLink(ctx context.Context, userID, linkID, tagID int64) error {
	// verify the link belongs to the user
	var ownerID int64
	err := s.db.QueryRow(ctx, `SELECT user_id FROM links WHERE id = $1`, linkID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		return errors.New("link not found or access denied")
	}
	// verify the tag also belongs to that user (previously another user's tag could be attached, leaking its name/color)
	var tagOwner int64
	if tagErr := s.db.QueryRow(ctx, `SELECT user_id FROM tags WHERE id = $1`, tagID).Scan(&tagOwner); tagErr != nil || tagOwner != userID {
		return errors.New("tag not found or access denied")
	}
	_, err = s.db.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, linkID, tagID)
	return err
}

// RemoveTagFromLink removes a link tag
func (s *TagService) RemoveTagFromLink(ctx context.Context, userID, linkID, tagID int64) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM link_tags WHERE link_id = $1 AND tag_id = $2 AND link_id IN (SELECT id FROM links WHERE user_id = $3)`,
		linkID, tagID, userID)
	return err
}
