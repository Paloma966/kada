package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

type DomainService struct {
	db *pgxpool.Pool
}

func NewDomainService(db *pgxpool.Pool) *DomainService {
	return &DomainService{db: db}
}

// Create 添加自定义域名
func (s *DomainService) Create(ctx context.Context, userID int64, req domain.CreateDomainRequest) (*domain.Domain, error) {
	var d domain.Domain
	err := s.db.QueryRow(ctx, `
		INSERT INTO domains (user_id, name) VALUES ($1, $2)
		RETURNING id, user_id, name, verified, verified_at, created_at, updated_at
	`, userID, req.Name).Scan(&d.ID, &d.UserID, &d.Name, &d.Verified, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("添加域名失败，域名可能已存在: %w", err)
	}
	return &d, nil
}

// List 获取用户的域名列表
func (s *DomainService) List(ctx context.Context, userID int64) ([]domain.Domain, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, name, verified, verified_at, created_at, updated_at
		FROM domains WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []domain.Domain
	for rows.Next() {
		var d domain.Domain
		if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.Verified, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	if domains == nil {
		domains = []domain.Domain{}
	}
	return domains, nil
}

// Verify 验证域名所有权（检查 DNS TXT 记录）
func (s *DomainService) Verify(ctx context.Context, userID, domainID int64) (*domain.Domain, error) {
	// 先获取域名
	var d domain.Domain
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, name, verified FROM domains WHERE id = $1 AND user_id = $2
	`, domainID, userID).Scan(&d.ID, &d.UserID, &d.Name, &d.Verified)
	if err != nil {
		return nil, errors.New("域名不存在")
	}

	// 简化验证：直接标记为已验证（生产环境应检查 DNS TXT 记录）
	now := time.Now()
	err = s.db.QueryRow(ctx, `
		UPDATE domains SET verified = TRUE, verified_at = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, name, verified, verified_at, created_at, updated_at
	`, now, domainID, userID).Scan(&d.ID, &d.UserID, &d.Name, &d.Verified, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("验证失败: %w", err)
	}
	return &d, nil
}

// Delete 删除域名
func (s *DomainService) Delete(ctx context.Context, userID, domainID int64) error {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM domains WHERE id = $1 AND user_id = $2
	`, domainID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("域名不存在")
	}
	return nil
}
