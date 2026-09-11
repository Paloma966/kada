package service

import (
	"context"
	"errors"
	"log"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

type WorkspaceService struct {
	db *pgxpool.Pool
}

func NewWorkspaceService(db *pgxpool.Pool) *WorkspaceService {
	return &WorkspaceService{db: db}
}

// Create creates a workspace
func (s *WorkspaceService) Create(ctx context.Context, userID int64, req domain.CreateWorkspaceRequest) (*domain.Workspace, error) {
	if !slugPattern.MatchString(req.Slug) {
		return nil, errors.New("invalid slug format: only lowercase letters, digits and hyphens are allowed, length 3-50")
	}

	var exists bool
	// fast-path check: on failure make no decision, the unique constraint is the final arbiter
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1)`, req.Slug).Scan(&exists); err == nil && exists {
		return nil, errors.New("that slug is already taken")
	}

	var w domain.Workspace
	err := s.db.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, user_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, user_id, created_at, updated_at
	`, req.Name, req.Slug, userID).Scan(
		&w.ID, &w.Name, &w.Slug, &w.UserID, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, errors.New("failed to create workspace")
	}

	return &w, nil
}

// List gets the user's workspace list
func (s *WorkspaceService) List(ctx context.Context, userID int64) ([]domain.Workspace, error) {
	rows, err := s.db.Query(ctx, `
		SELECT w.id, w.name, w.slug, w.user_id, w.created_at, w.updated_at,
		       COALESCE((SELECT COUNT(*) FROM links WHERE workspace_id = w.id), 0) as link_count
		FROM workspaces w
		WHERE w.user_id = $1
		ORDER BY w.created_at DESC
	`, userID)
	if err != nil {
		return nil, errors.New("failed to list workspaces")
	}
	defer rows.Close()

	var workspaces []domain.Workspace
	for rows.Next() {
		var w domain.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.UserID, &w.CreatedAt, &w.UpdatedAt, &w.LinkCount); err != nil {
			return nil, errors.New("failed to list workspaces")
		}
		workspaces = append(workspaces, w)
	}
	if workspaces == nil {
		workspaces = []domain.Workspace{}
	}

	return workspaces, nil
}

// GetByID gets a workspace by ID
func (s *WorkspaceService) GetByID(ctx context.Context, workspaceID, userID int64) (*domain.Workspace, error) {
	var w domain.Workspace
	err := s.db.QueryRow(ctx, `
		SELECT w.id, w.name, w.slug, w.user_id, w.created_at, w.updated_at,
		       COALESCE((SELECT COUNT(*) FROM links WHERE workspace_id = w.id), 0) as link_count
		FROM workspaces w
		WHERE w.id = $1 AND w.user_id = $2
	`, workspaceID, userID).Scan(
		&w.ID, &w.Name, &w.Slug, &w.UserID, &w.CreatedAt, &w.UpdatedAt, &w.LinkCount,
	)
	if err != nil {
		return nil, errors.New("workspace not found")
	}

	return &w, nil
}

// Update updates a workspace
func (s *WorkspaceService) Update(ctx context.Context, workspaceID, userID int64, req domain.UpdateWorkspaceRequest) (*domain.Workspace, error) {
	if req.Slug != nil {
		if !slugPattern.MatchString(*req.Slug) {
			return nil, errors.New("invalid slug format")
		}
		var exists bool
		// fast-path check: on failure make no decision, the unique constraint is the final arbiter
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1 AND id != $2)`, *req.Slug, workspaceID).Scan(&exists); err == nil && exists {
			return nil, errors.New("that slug is already taken")
		}
	}

	var w domain.Workspace
	err := s.db.QueryRow(ctx, `
		UPDATE workspaces SET
			name = COALESCE($3, name),
			slug = COALESCE($4, slug),
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, name, slug, user_id, created_at, updated_at
	`, workspaceID, userID, req.Name, req.Slug).Scan(
		&w.ID, &w.Name, &w.Slug, &w.UserID, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, errors.New("workspace not found or access denied")
	}

	return &w, nil
}

// Delete deletes a workspace
func (s *WorkspaceService) Delete(ctx context.Context, workspaceID, userID int64) error {
	// unlink associated links first
	if _, err := s.db.Exec(ctx, `UPDATE links SET workspace_id = NULL WHERE workspace_id = $1 AND user_id = $2`, workspaceID, userID); err != nil {
		log.Printf("unlink workspace %d failed: %v", workspaceID, err)
	}

	_, err := s.db.Exec(ctx, `DELETE FROM workspaces WHERE id = $1 AND user_id = $2`, workspaceID, userID)
	if err != nil {
		return errors.New("failed to delete workspace")
	}
	return nil
}
