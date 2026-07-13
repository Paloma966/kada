package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

type FolderService struct {
	db *pgxpool.Pool
}

func NewFolderService(db *pgxpool.Pool) *FolderService {
	return &FolderService{db: db}
}

func (s *FolderService) Create(ctx context.Context, userID int64, req domain.CreateFolderRequest) (*domain.Folder, error) {
	var f domain.Folder
	err := s.db.QueryRow(ctx,
		`INSERT INTO folders (user_id, name) VALUES ($1, $2) RETURNING id, user_id, name, created_at, updated_at`,
		userID, req.Name,
	).Scan(&f.ID, &f.UserID, &f.Name, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, errors.New("创建文件夹失败")
	}
	return &f, nil
}

func (s *FolderService) List(ctx context.Context, userID int64) ([]domain.Folder, error) {
	rows, err := s.db.Query(ctx,
		`SELECT f.id, f.user_id, f.name, f.created_at, f.updated_at, COUNT(l.id) as link_count
		FROM folders f LEFT JOIN links l ON f.id = l.folder_id
		WHERE f.user_id = $1 GROUP BY f.id ORDER BY f.name`, userID)
	if err != nil {
		return nil, errors.New("查询文件夹列表失败")
	}
	defer rows.Close()

	var folders []domain.Folder
	for rows.Next() {
		var f domain.Folder
		rows.Scan(&f.ID, &f.UserID, &f.Name, &f.CreatedAt, &f.UpdatedAt, &f.LinkCount)
		folders = append(folders, f)
	}
	if folders == nil {
		folders = []domain.Folder{}
	}
	return folders, nil
}

func (s *FolderService) Update(ctx context.Context, userID, folderID int64, name string) (*domain.Folder, error) {
	var f domain.Folder
	err := s.db.QueryRow(ctx,
		`UPDATE folders SET name = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3 RETURNING id, user_id, name, created_at, updated_at`,
		name, folderID, userID,
	).Scan(&f.ID, &f.UserID, &f.Name, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, errors.New("文件夹不存在或无权限")
	}
	return &f, nil
}

func (s *FolderService) Delete(ctx context.Context, userID, folderID int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM folders WHERE id = $1 AND user_id = $2`, folderID, userID)
	if err != nil {
		return errors.New("删除文件夹失败")
	}
	return nil
}
