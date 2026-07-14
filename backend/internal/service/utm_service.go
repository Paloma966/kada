package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

type UTMTemplateService struct {
	db *pgxpool.Pool
}

func NewUTMTemplateService(db *pgxpool.Pool) *UTMTemplateService {
	return &UTMTemplateService{db: db}
}

func (s *UTMTemplateService) Create(ctx context.Context, userID int64, req domain.CreateUTMTemplateRequest) (*domain.UTMTemplate, error) {
	var t domain.UTMTemplate
	err := s.db.QueryRow(ctx, `
		INSERT INTO utm_templates (user_id, name, utm_source, utm_medium, utm_campaign, utm_term, utm_content)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, user_id, name, utm_source, utm_medium, utm_campaign, utm_term, utm_content, created_at, updated_at
	`, userID, req.Name, req.UTMSource, req.UTMMedium, req.UTMCampaign, req.UTMTerm, req.UTMContent).
		Scan(&t.ID, &t.UserID, &t.Name, &t.UTMSource, &t.UTMMedium, &t.UTMCampaign, &t.UTMTerm, &t.UTMContent, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}
	return &t, nil
}

func (s *UTMTemplateService) List(ctx context.Context, userID int64) ([]domain.UTMTemplate, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, name, utm_source, utm_medium, utm_campaign, utm_term, utm_content, created_at, updated_at
		FROM utm_templates WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []domain.UTMTemplate
	for rows.Next() {
		var t domain.UTMTemplate
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.UTMSource, &t.UTMMedium, &t.UTMCampaign, &t.UTMTerm, &t.UTMContent, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []domain.UTMTemplate{}
	}
	return templates, nil
}

func (s *UTMTemplateService) Delete(ctx context.Context, userID, templateID int64) error {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM utm_templates WHERE id = $1 AND user_id = $2
	`, templateID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("模板不存在")
	}
	return nil
}
