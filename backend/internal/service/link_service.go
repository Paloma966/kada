package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

type LinkService struct {
	db      *pgxpool.Pool
	baseURL string
}

func NewLinkService(db *pgxpool.Pool, baseURL string) *LinkService {
	return &LinkService{db: db, baseURL: baseURL}
}

// Create 创建短链接
func (s *LinkService) Create(ctx context.Context, userID int64, req domain.CreateLinkRequest) (*domain.LinkInfo, error) {
	shortCode := generateShortCode()
	domain_ := "kada.link"
	if req.Domain != nil && *req.Domain != "" {
		domain_ = *req.Domain
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	var passwordHash *string
	if req.Password != nil && *req.Password != "" {
		hash := hashPassword(*req.Password)
		passwordHash = &hash
	}

	var info domain.LinkInfo
	err := s.db.QueryRow(ctx, `
		INSERT INTO links (short_code, original_url, title, description, image_url, domain, password_hash, expires_at, user_id, workspace_id, utm_source, utm_medium, utm_campaign, utm_term, utm_content, ios_url, android_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id, short_code, original_url, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), domain, click_count, is_active, expires_at, created_at, updated_at
	`,
		shortCode, req.OriginalURL, req.Title, req.Description, req.ImageURL,
		domain_, passwordHash, expiresAt, userID, req.WorkspaceID,
		req.UTMSource, req.UTMMedium, req.UTMCampaign, req.UTMTerm, req.UTMContent,
		req.IosURL, req.AndroidURL,
	).Scan(
		&info.ID, &info.ShortCode, &info.OriginalURL, &info.Title, &info.Description,
		&info.ImageURL, &info.Domain, &info.ClickCount, &info.IsActive,
		&info.ExpiresAt, &info.CreatedAt, &info.UpdatedAt,
	)
	if err != nil {
		return nil, errors.New("创建短链接失败: " + err.Error())
	}

	info.ShortURL = s.buildShortURL(info.Domain, info.ShortCode)
	return &info, nil
}

// GetByCode 根据短码获取链接
func (s *LinkService) GetByCode(ctx context.Context, shortCode string) (*domain.LinkInfo, error) {
	var info domain.LinkInfo
	err := s.db.QueryRow(ctx, `
		SELECT id, short_code, original_url, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), domain, click_count, is_active, expires_at, created_at, updated_at
		FROM links WHERE short_code = $1 AND is_active = TRUE
	`, shortCode).Scan(
		&info.ID, &info.ShortCode, &info.OriginalURL, &info.Title, &info.Description,
		&info.ImageURL, &info.Domain, &info.ClickCount, &info.IsActive,
		&info.ExpiresAt, &info.CreatedAt, &info.UpdatedAt,
	)
	if err != nil {
		return nil, errors.New("链接不存在或已失效")
	}

	// 检查是否过期
	if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("链接已过期")
	}

	info.ShortURL = s.buildShortURL(info.Domain, info.ShortCode)
	return &info, nil
}

// HasPassword 检查链接是否设置了密码
func (s *LinkService) HasPassword(ctx context.Context, shortCode string) bool {
	var passwordHash *string
	err := s.db.QueryRow(ctx, `
		SELECT password_hash FROM links WHERE short_code = $1
	`, shortCode).Scan(&passwordHash)
	if err != nil || passwordHash == nil || *passwordHash == "" {
		return false
	}
	return true
}

// CheckPassword 检查链接密码
func (s *LinkService) CheckPassword(ctx context.Context, shortCode, password string) (bool, *domain.LinkInfo, error) {
	var passwordHash *string
	var info domain.LinkInfo
	err := s.db.QueryRow(ctx, `
		SELECT id, short_code, original_url, COALESCE(title,''), COALESCE(description,''), domain, click_count, is_active, expires_at, password_hash, created_at, updated_at
		FROM links WHERE short_code = $1 AND is_active = TRUE
	`, shortCode).Scan(
		&info.ID, &info.ShortCode, &info.OriginalURL, &info.Title, &info.Description,
		&info.Domain, &info.ClickCount, &info.IsActive,
		&info.ExpiresAt, &passwordHash, &info.CreatedAt, &info.UpdatedAt,
	)
	if err != nil {
		return false, nil, errors.New("链接不存在或已失效")
	}

	if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
		return false, nil, errors.New("链接已过期")
	}

	if passwordHash == nil || *passwordHash == "" {
		return true, &info, nil // 无密码
	}

	// 简单密码验证（生产环境应使用 bcrypt）
	if password == "" || !checkPasswordHash(password, *passwordHash) {
		return false, &info, nil
	}

	info.ShortURL = s.buildShortURL(info.Domain, info.ShortCode)
	return true, &info, nil
}

// List 获取用户链接列表
func (s *LinkService) List(ctx context.Context, userID int64, page, pageSize int, search string, folderID, tagID int64) (*domain.PaginatedLinks, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 动态构建查询条件
	where := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIdx := 2

	if search != "" {
		where += " AND (title ILIKE $" + strconv.Itoa(argIdx) + " OR original_url ILIKE $" + strconv.Itoa(argIdx) + " OR short_code ILIKE $" + strconv.Itoa(argIdx) + ")"
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if folderID > 0 {
		where += " AND folder_id = $" + strconv.Itoa(argIdx)
		args = append(args, folderID)
		argIdx++
	}
	if tagID > 0 {
		where += " AND id IN (SELECT link_id FROM link_tags WHERE tag_id = $" + strconv.Itoa(argIdx) + ")"
		args = append(args, tagID)
		argIdx++
	}

	// 计数
	var total int64
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM links `+where, args...).Scan(&total)

	// 查询
	query := `SELECT id, short_code, original_url, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), domain, click_count, is_active, expires_at, created_at, updated_at
		FROM links ` + where + ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.New("查询链接列表失败")
	}
	defer rows.Close()

	var links []domain.LinkInfo
	for rows.Next() {
		var l domain.LinkInfo
		rows.Scan(&l.ID, &l.ShortCode, &l.OriginalURL, &l.Title, &l.Description,
			&l.ImageURL, &l.Domain, &l.ClickCount, &l.IsActive,
			&l.ExpiresAt, &l.CreatedAt, &l.UpdatedAt)
		l.ShortURL = s.buildShortURL(l.Domain, l.ShortCode)
		links = append(links, l)
	}

	return &domain.PaginatedLinks{
		Links: links, TotalCount: total, Page: page, PageSize: pageSize,
	}, nil
}

// Update 更新链接
func (s *LinkService) Update(ctx context.Context, linkID, userID int64, req domain.UpdateLinkRequest) (*domain.LinkInfo, error) {
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	var passwordHash *string
	if req.Password != nil {
		hash := hashPassword(*req.Password)
		passwordHash = &hash
	}

	var info domain.LinkInfo
	err := s.db.QueryRow(ctx, `
		UPDATE links SET
			original_url = COALESCE($2, original_url),
			title = COALESCE($3, title),
			description = COALESCE($4, description),
			image_url = COALESCE($5, image_url),
			password_hash = COALESCE($6, password_hash),
			expires_at = COALESCE($7, expires_at),
			is_active = COALESCE($8, is_active),
			updated_at = NOW()
		WHERE id = $1 AND user_id = $9
		RETURNING id, short_code, original_url, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), domain, click_count, is_active, expires_at, created_at, updated_at
	`,
		linkID, req.OriginalURL, req.Title, req.Description, req.ImageURL,
		passwordHash, expiresAt, req.IsActive, userID,
	).Scan(
		&info.ID, &info.ShortCode, &info.OriginalURL, &info.Title, &info.Description,
		&info.ImageURL, &info.Domain, &info.ClickCount, &info.IsActive,
		&info.ExpiresAt, &info.CreatedAt, &info.UpdatedAt,
	)
	if err != nil {
		return nil, errors.New("更新链接失败，链接不存在或无权限")
	}

	info.ShortURL = s.buildShortURL(info.Domain, info.ShortCode)
	return &info, nil
}

// Delete 删除链接
func (s *LinkService) Delete(ctx context.Context, linkID, userID int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM links WHERE id = $1 AND user_id = $2`, linkID, userID)
	if err != nil {
		return errors.New("删除链接失败")
	}
	return nil
}

// LogClick 记录点击
func (s *LinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
	s.db.Exec(ctx, `
		INSERT INTO click_logs (link_id, ip, user_agent, platform, referer)
		VALUES ($1, $2, $3, $4, $5)
	`, linkID, ip, userAgent, platform, referer)

	s.db.Exec(ctx, `UPDATE links SET click_count = click_count + 1 WHERE id = $1`, linkID)
}

// buildShortURL 构建完整短链接
func (s *LinkService) buildShortURL(domain, code string) string {
	return "https://" + domain + "/r/" + code
}

// generateShortCode 生成6字节随机短码
func generateShortCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// hashPassword 简单哈希（可替换为 bcrypt）
func hashPassword(pwd string) string {
	// 简单的 SHA256 哈希（生产环境应使用 bcrypt）
	h := sha256.Sum256([]byte(pwd))
	return hex.EncodeToString(h[:])
}

func checkPasswordHash(password, hash string) bool {
	return hashPassword(password) == hash
}
