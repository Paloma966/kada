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

// Create creates an API Token and returns the raw token (this is the only time it is returned)
func (s *APITokenService) Create(ctx context.Context, userID int64, req domain.CreateAPITokenRequest) (*domain.CreateAPITokenResponse, error) {
	// generate a random token
	b := make([]byte, 24)
	rand.Read(b)
	rawToken := "kada_" + hex.EncodeToString(b)

	// store the SHA256 hash
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
		return nil, errors.New("failed to create API token")
	}

	return &domain.CreateAPITokenResponse{
		Token:    rawToken,
		APIToken: info,
	}, nil
}

// List lists the user's API Tokens
func (s *APITokenService) List(ctx context.Context, userID int64) ([]domain.APIToken, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, name, last_used, created_at
		FROM api_tokens WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, errors.New("failed to list API tokens")
	}
	defer rows.Close()

	var tokens []domain.APIToken
	for rows.Next() {
		var t domain.APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.LastUsed, &t.CreatedAt); err != nil {
			return nil, errors.New("failed to list API tokens")
		}
		tokens = append(tokens, t)
	}
	if tokens == nil {
		tokens = []domain.APIToken{}
	}
	return tokens, nil
}

// Delete deletes an API Token
func (s *APITokenService) Delete(ctx context.Context, id, userID int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return errors.New("failed to delete API token")
	}
	return nil
}

// ValidateToken validates an API Token and returns the userID
func (s *APITokenService) ValidateToken(ctx context.Context, rawToken string) (int64, error) {
	if rawToken == "" || len(rawToken) < 6 || rawToken[:5] != "kada_" {
		return 0, fmt.Errorf("invalid API token format")
	}

	tokenHash := sha256Hex(rawToken)

	var userID int64
	err := s.db.QueryRow(ctx, `
		UPDATE api_tokens SET last_used = NOW()
		WHERE token_hash = $1
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("invalid API token")
	}

	return userID, nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
