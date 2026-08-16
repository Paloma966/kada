package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

// txtPrefix 域名所有权验证的 DNS TXT 记录前缀
const txtPrefix = "kada-verify="

// verificationCode 计算域名验证码：sha256(userID:domainID:name) 前 16 位 hex
func verificationCode(userID, domainID int64, name string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s", userID, domainID, name)))
	return hex.EncodeToString(h[:])[:16]
}

// expectedTXT 期望用户在 DNS 中配置的 TXT 记录值
func expectedTXT(userID, domainID int64, name string) string {
	return txtPrefix + verificationCode(userID, domainID, name)
}

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
		return nil, errors.New("添加域名失败，域名可能已存在")
	}
	d.VerificationCode = verificationCode(userID, d.ID, d.Name)
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
		if !d.Verified {
			d.VerificationCode = verificationCode(d.UserID, d.ID, d.Name)
		}
		domains = append(domains, d)
	}
	if domains == nil {
		domains = []domain.Domain{}
	}
	return domains, nil
}

// Verify 验证域名所有权：检查 DNS TXT 记录
func (s *DomainService) Verify(ctx context.Context, userID, domainID int64) (*domain.Domain, error) {
	var d domain.Domain
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, name, verified FROM domains WHERE id = $1 AND user_id = $2
	`, domainID, userID).Scan(&d.ID, &d.UserID, &d.Name, &d.Verified)
	if err != nil {
		return nil, errors.New("域名不存在")
	}

	want := expectedTXT(userID, domainID, d.Name)

	// DNS TXT 查询（8 秒超时）
	dnsCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	records, err := net.DefaultResolver.LookupTXT(dnsCtx, d.Name)
	if err != nil {
		return nil, fmt.Errorf("DNS 查询失败（%s），请确认域名已正确解析", d.Name)
	}
	found := false
	for _, r := range records {
		if strings.TrimSpace(r) == want {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("未找到验证记录，请在域名 %s 的 DNS 中添加 TXT 记录：%s", d.Name, want)
	}

	now := time.Now()
	err = s.db.QueryRow(ctx, `
		UPDATE domains SET verified = TRUE, verified_at = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id, user_id, name, verified, verified_at, created_at, updated_at
	`, now, domainID, userID).Scan(&d.ID, &d.UserID, &d.Name, &d.Verified, &d.VerifiedAt, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, errors.New("验证失败")
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
