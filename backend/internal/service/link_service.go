package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/domain"
)

// shortCodePattern 短码只允许字母、数字、下划线和连字符
var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{4,20}$`)

type LinkService struct {
	db      *pgxpool.Pool
	baseURL string
}

func NewLinkService(db *pgxpool.Pool, baseURL string) *LinkService {
	return &LinkService{db: db, baseURL: baseURL}
}

// Create 创建短链接
func (s *LinkService) Create(ctx context.Context, userID int64, req domain.CreateLinkRequest) (*domain.LinkInfo, error) {
	var shortCode string

	if req.ShortCode != nil && *req.ShortCode != "" {
		if !shortCodePattern.MatchString(*req.ShortCode) {
			return nil, errors.New("短码格式无效：只允许字母、数字、下划线和连字符，长度4-20位")
		}
		var exists bool
		s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1)`, *req.ShortCode).Scan(&exists)
		if exists {
			return nil, errors.New("该短码已被占用，请换一个")
		}
		shortCode = *req.ShortCode
	} else {
		shortCode = generateShortCode()
	}

	domain_ := "kada.click"
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
		INSERT INTO links (short_code, original_url, title, description, image_url, domain, password_hash, expires_at, user_id, workspace_id, folder_id, utm_source, utm_medium, utm_campaign, utm_term, utm_content, ios_url, android_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING id, short_code, original_url, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), domain, click_count, is_active, expires_at, created_at, updated_at
	`,
		shortCode, req.OriginalURL, req.Title, req.Description, req.ImageURL,
		domain_, passwordHash, expiresAt, userID, req.WorkspaceID, req.FolderID,
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

	// 关联标签
	for _, tagID := range req.TagIDs {
		s.db.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, info.ID, tagID)
	}

	info.ShortURL = s.buildShortURL(info.Domain, info.ShortCode)
	return &info, nil
}

// GetByID 根据ID获取链接（含文件夹、标签、UTM等完整信息）
func (s *LinkService) GetByID(ctx context.Context, linkID, userID int64) (*domain.LinkInfo, error) {
	var info domain.LinkInfo
	err := s.db.QueryRow(ctx, `
		SELECT l.id, l.short_code, l.original_url, COALESCE(l.title,''), COALESCE(l.description,''),
		       COALESCE(l.image_url,''), l.domain, l.click_count, l.is_active, l.expires_at,
		       l.created_at, l.updated_at, l.folder_id,
		       l.utm_source, l.utm_medium, l.utm_campaign, l.utm_term, l.utm_content,
		       l.ios_url, l.android_url, l.password_hash
		FROM links l WHERE l.id = $1 AND l.user_id = $2
	`, linkID, userID).Scan(
		&info.ID, &info.ShortCode, &info.OriginalURL, &info.Title, &info.Description,
		&info.ImageURL, &info.Domain, &info.ClickCount, &info.IsActive,
		&info.ExpiresAt, &info.CreatedAt, &info.UpdatedAt, &info.FolderID,
		&info.UTMSource, &info.UTMMedium, &info.UTMCampaign, &info.UTMTerm, &info.UTMContent,
		&info.IosURL, &info.AndroidURL, &info.PasswordHash,
	)
	if err != nil {
		return nil, errors.New("链接不存在或已失效")
	}

	// 查询文件夹名
	if info.FolderID != nil {
		var folderName string
		err := s.db.QueryRow(ctx, `SELECT name FROM folders WHERE id = $1`, *info.FolderID).Scan(&folderName)
		if err == nil {
			info.FolderName = &folderName
		}
	}

	// 查询标签
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.name, t.color FROM tags t
		JOIN link_tags lt ON t.id = lt.tag_id
		WHERE lt.link_id = $1
	`, linkID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t domain.LinkTagInfo
			rows.Scan(&t.ID, &t.Name, &t.Color)
			info.Tags = append(info.Tags, t)
		}
	}
	if info.Tags == nil {
		info.Tags = []domain.LinkTagInfo{}
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

	if password == "" || !checkPasswordHash(password, *passwordHash) {
		return false, &info, nil
	}

	info.ShortURL = s.buildShortURL(info.Domain, info.ShortCode)
	return true, &info, nil
}

// List 获取用户链接列表
func (s *LinkService) List(ctx context.Context, userID int64, page, pageSize int, search string, folderID, tagID int64, sort string) (*domain.PaginatedLinks, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	where := "WHERE l.user_id = $1"
	args := []interface{}{userID}
	argIdx := 2

	if search != "" {
		where += " AND (l.title ILIKE $" + strconv.Itoa(argIdx) + " OR l.original_url ILIKE $" + strconv.Itoa(argIdx) + " OR l.short_code ILIKE $" + strconv.Itoa(argIdx) + ")"
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if folderID > 0 {
		where += " AND l.folder_id = $" + strconv.Itoa(argIdx)
		args = append(args, folderID)
		argIdx++
	}
	if tagID > 0 {
		where += " AND l.id IN (SELECT link_id FROM link_tags WHERE tag_id = $" + strconv.Itoa(argIdx) + ")"
		args = append(args, tagID)
		argIdx++
	}

	var total int64
	s.db.QueryRow(ctx, `SELECT COUNT(*) FROM links l `+where, args...).Scan(&total)

	orderBy := "l.created_at DESC"
	switch sort {
	case "clicks_desc":
		orderBy = "l.click_count DESC"
	case "clicks_asc":
		orderBy = "l.click_count ASC"
	case "created_asc":
		orderBy = "l.created_at ASC"
	}

	query := `SELECT l.id, l.short_code, l.original_url, COALESCE(l.title,''), COALESCE(l.description,''), COALESCE(l.image_url,''), l.domain, l.click_count, l.is_active, l.expires_at, l.created_at, l.updated_at, l.folder_id
		FROM links l ` + where + ` ORDER BY ` + orderBy + ` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
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
			&l.ExpiresAt, &l.CreatedAt, &l.UpdatedAt, &l.FolderID)
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

	// 校验自定义短码
	if req.ShortCode != nil && *req.ShortCode != "" {
		if !shortCodePattern.MatchString(*req.ShortCode) {
			return nil, errors.New("短码格式无效：只允许字母、数字、下划线和连字符，长度4-20位")
		}
		var exists bool
		s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1 AND id != $2)`, *req.ShortCode, linkID).Scan(&exists)
		if exists {
			return nil, errors.New("该短码已被占用，请换一个")
		}
	}

	var info domain.LinkInfo
	err := s.db.QueryRow(ctx, `
		UPDATE links SET
			original_url = COALESCE($2, original_url),
			short_code = COALESCE($3, short_code),
			title = COALESCE($4, title),
			description = COALESCE($5, description),
			image_url = COALESCE($6, image_url),
			domain = COALESCE($7, domain),
			password_hash = COALESCE($8, password_hash),
			expires_at = COALESCE($9, expires_at),
			is_active = COALESCE($10, is_active),
			folder_id = COALESCE($11, folder_id),
			utm_source = COALESCE($12, utm_source),
			utm_medium = COALESCE($13, utm_medium),
			utm_campaign = COALESCE($14, utm_campaign),
			utm_term = COALESCE($15, utm_term),
			utm_content = COALESCE($16, utm_content),
			ios_url = COALESCE($17, ios_url),
			android_url = COALESCE($18, android_url),
			updated_at = NOW()
		WHERE id = $1 AND user_id = $19
		RETURNING id, short_code, original_url, COALESCE(title,''), COALESCE(description,''), COALESCE(image_url,''), domain, click_count, is_active, expires_at, folder_id, created_at, updated_at
	`,
		linkID, req.OriginalURL, req.ShortCode, req.Title, req.Description, req.ImageURL,
		req.Domain, passwordHash, expiresAt, req.IsActive, req.FolderID,
		req.UTMSource, req.UTMMedium, req.UTMCampaign, req.UTMTerm, req.UTMContent,
		req.IosURL, req.AndroidURL, userID,
	).Scan(
		&info.ID, &info.ShortCode, &info.OriginalURL, &info.Title, &info.Description,
		&info.ImageURL, &info.Domain, &info.ClickCount, &info.IsActive,
		&info.ExpiresAt, &info.FolderID, &info.CreatedAt, &info.UpdatedAt,
	)
	if err != nil {
		return nil, errors.New("更新链接失败，链接不存在或无权限")
	}

	// 更新标签关联
	if req.TagIDs != nil {
		s.db.Exec(ctx, `DELETE FROM link_tags WHERE link_id = $1`, linkID)
		for _, tagID := range req.TagIDs {
			s.db.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, linkID, tagID)
		}
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

// generateShortCode 生成4字节随机短码（8位十六进制）
func generateShortCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// hashPassword 简单哈希（可替换为 bcrypt）
func hashPassword(pwd string) string {
	h := sha256.Sum256([]byte(pwd))
	return hex.EncodeToString(h[:])
}

func checkPasswordHash(password, hash string) bool {
	return hashPassword(password) == hash
}
