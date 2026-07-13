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
		return nil, errors.New("创建标签失败")
	}
	return &t, nil
}

func (s *TagService) List(ctx context.Context, userID int64) ([]domain.Tag, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, name, color, created_at FROM tags WHERE user_id = $1 ORDER BY name`, userID)
	if err != nil {
		return nil, errors.New("查询标签列表失败")
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Color, &t.CreatedAt)
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
		return errors.New("删除标签失败")
	}
	return nil
}

// AddTagToLink 为链接添加标签
func (s *TagService) AddTagToLink(ctx context.Context, userID, linkID, tagID int64) error {
	// 验证链接属于用户
	var ownerID int64
	err := s.db.QueryRow(ctx, `SELECT user_id FROM links WHERE id = $1`, linkID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		return errors.New("链接不存在或无权限")
	}
	_, err = s.db.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, linkID, tagID)
	return err
}

// RemoveTagFromLink 移除链接标签
func (s *TagService) RemoveTagFromLink(ctx context.Context, userID, linkID, tagID int64) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM link_tags WHERE link_id = $1 AND tag_id = $2 AND link_id IN (SELECT id FROM links WHERE user_id = $3)`,
		linkID, tagID, userID)
	return err
}
