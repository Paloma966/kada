package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

type APITokenService struct {
	db *pgxpool.Pool
}

func NewAPITokenService(db *pgxpool.Pool) *APITokenService {
	return &APITokenService{db: db}
}

// Create 创建 API Token，返回原始 token（仅此一次）
func (s *APITokenService) Create(ctx context.Context, userID int64, req domain.CreateAPITokenRequest) (*domain.CreateAPITokenResponse, error) {
	// 生成随机 token
	b := make([]byte, 24)
	rand.Read(b)
	rawToken := "kada_" + hex.EncodeToString(b)

	// 存储 SHA256 哈希
	tokenHash := sha256Hex(rawToken)

	var info domain.APIToken
	err := s.db.QueryRow(ctx, `
		INSERT INTO api_tokens (user_id, name, token_hash)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, name, last_used, created_at
	`, userID, req.Name, tokenHash).Scan(
		&info.ID, &info.UserID, &info.Name, &info.LastUsed, &info.CreatedAt,
	)
	if err != nil {
		return nil, errors.New("创建 API Token 失败")
	}

	return &domain.CreateAPITokenResponse{
		Token:    rawToken,
		APIToken: info,
	}, nil
}

// List 列出用户的 API Tokens
func (s *APITokenService) List(ctx context.Context, userID int64) ([]domain.APIToken, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, name, last_used, created_at
		FROM api_tokens WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, errors.New("查询 API Token 列表失败")
	}
	defer rows.Close()

	var tokens []domain.APIToken
	for rows.Next() {
		var t domain.APIToken
		rows.Scan(&t.ID, &t.UserID, &t.Name, &t.LastUsed, &t.CreatedAt)
		tokens = append(tokens, t)
	}
	if tokens == nil {
		tokens = []domain.APIToken{}
	}
	return tokens, nil
}

// Delete 删除 API Token
func (s *APITokenService) Delete(ctx context.Context, id, userID int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return errors.New("删除 API Token 失败")
	}
	return nil
}

// ValidateToken 验证 API Token，返回 userID
func (s *APITokenService) ValidateToken(ctx context.Context, rawToken string) (int64, error) {
	if rawToken == "" || len(rawToken) < 6 || rawToken[:5] != "kada_" {
		return 0, fmt.Errorf("无效的 API Token 格式")
	}

	tokenHash := sha256Hex(rawToken)

	var userID int64
	err := s.db.QueryRow(ctx, `
		UPDATE api_tokens SET last_used = NOW()
		WHERE token_hash = $1
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("无效的 API Token")
	}

	return userID, nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
