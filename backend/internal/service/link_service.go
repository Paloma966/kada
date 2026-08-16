package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/infra/urlcheck"
	"github.com/chun/kada-backend/internal/mq"
)

// shortCodePattern 短码只允许字母、数字、下划线和连字符
var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{4,20}$`)

type LinkService struct {
	db          *pgxpool.Pool
	baseURL     string
	cache       *CacheService
	kafka       mq.ClickPublisher // Kafka 发布者；nil 表示禁用
	clickWriter ClickWriter       // 直写（降级回退用）
}

func NewLinkService(db *pgxpool.Pool, baseURL string, cache *CacheService, kafka mq.ClickPublisher, clickWriter ClickWriter) *LinkService {
	return &LinkService{db: db, baseURL: baseURL, cache: cache, kafka: kafka, clickWriter: clickWriter}
}

// Create 创建短链接
func (s *LinkService) Create(ctx context.Context, userID int64, req domain.CreateLinkRequest) (*domain.LinkInfo, error) {
	if !urlcheck.IsSafeTarget(req.OriginalURL) {
		return nil, errors.New("目标链接仅支持 http/https 协议")
	}

	// 校验文件夹/工作区归属（此前任意 ID 可挂接，跨用户泄漏名称）
	if err := s.validateOwnedRefs(ctx, userID, req.FolderID, req.WorkspaceID); err != nil {
		return nil, err
	}

	var shortCode string

	if req.ShortCode != nil && *req.ShortCode != "" {
		if !shortCodePattern.MatchString(*req.ShortCode) {
			return nil, errors.New("短码格式无效：只允许字母、数字、下划线和连字符，长度4-20位")
		}
		var exists bool
		// 快速路径检查：失败时不做判断，最终由唯一约束仲裁
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1)`, *req.ShortCode).Scan(&exists); err == nil && exists {
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
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, errors.New("过期时间格式无效，请使用 RFC3339 格式（如 2025-01-01T00:00:00Z）")
		}
		expiresAt = &t
	}

	var passwordHash *string
	if req.Password != nil && *req.Password != "" {
		hash := hashPassword(*req.Password)
		passwordHash = &hash
	}

	var info domain.LinkInfo
	// 唯一约束是短码冲突的最终仲裁：检查-插入竞态下 INSERT 会报 23505，
	// 自定义短码返回友好错误，随机短码换码重试
	for attempt := 0; ; attempt++ {
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
		if err == nil {
			break
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if req.ShortCode != nil && *req.ShortCode != "" {
				return nil, errors.New("该短码已被占用，请换一个")
			}
			if attempt >= 4 {
				log.Printf("create link failed: %v", err)
				return nil, errors.New("生成短码失败，请重试")
			}
			shortCode = generateShortCode()
			continue
		}
		log.Printf("create link failed: %v", err)
		return nil, errors.New("创建短链接失败")
	}

	// 关联标签（仅允许挂接自己的标签，防止跨用户泄漏标签名/颜色）
	for _, tagID := range req.TagIDs {
		var tagOwner int64
		if err := s.db.QueryRow(ctx, `SELECT user_id FROM tags WHERE id = $1`, tagID).Scan(&tagOwner); err != nil || tagOwner != userID {
			log.Printf("skip non-owned tag %d for link %d (user %d)", tagID, info.ID, userID)
			continue
		}
		if _, err := s.db.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, info.ID, tagID); err != nil {
			log.Printf("attach tag %d to link %d failed: %v", tagID, info.ID, err)
		}
	}

	info.ShortURL = s.BuildShortURL(info.Domain, info.ShortCode)

	// 写入缓存
	if s.cache != nil {
		s.cache.SetLink(ctx, &info)
	}

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
		return nil, domain.ErrLinkNotFound
	}

	// 查询文件夹名
	if info.FolderID != nil {
		var folderName string
		folderErr := s.db.QueryRow(ctx, `SELECT name FROM folders WHERE id = $1`, *info.FolderID).Scan(&folderName)
		if folderErr == nil {
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
			if err := rows.Scan(&t.ID, &t.Name, &t.Color); err != nil {
				log.Printf("scan link tags failed: %v", err)
				continue
			}
			info.Tags = append(info.Tags, t)
		}
	}
	if info.Tags == nil {
		info.Tags = []domain.LinkTagInfo{}
	}

	info.ShortURL = s.BuildShortURL(info.Domain, info.ShortCode)
	return &info, nil
}

// GetByCode 根据短码获取链接（优先从缓存读取）
func (s *LinkService) GetByCode(ctx context.Context, shortCode string) (*domain.LinkInfo, error) {
	// 尝试从缓存获取
	if s.cache != nil {
		if info, ok := s.cache.GetLink(ctx, shortCode); ok {
			// 检查是否过期
			if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
				s.cache.InvalidateLink(ctx, shortCode)
				return nil, errors.New("链接已过期")
			}
			return info, nil
		}
	}

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
		return nil, domain.ErrLinkNotFound
	}

	// 检查是否过期
	if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("链接已过期")
	}

	info.ShortURL = s.BuildShortURL(info.Domain, info.ShortCode)

	// 写入缓存
	if s.cache != nil {
		s.cache.SetLink(ctx, &info)
	}

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
		return false, nil, domain.ErrLinkNotFound
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

	info.ShortURL = s.BuildShortURL(info.Domain, info.ShortCode)
	return true, &info, nil
}

// List 获取用户链接列表
func (s *LinkService) List(ctx context.Context, userID int64, page, pageSize int, search string, folderID, tagID, workspaceID int64, sort string) (*domain.PaginatedLinks, error) {
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
	if workspaceID > 0 {
		where += " AND l.workspace_id = $" + strconv.Itoa(argIdx)
		args = append(args, workspaceID)
		argIdx++
	}

	var total int64
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM links l `+where, args...).Scan(&total); err != nil {
		log.Printf("count links failed: %v", err)
		total = 0
	}

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
		if err := rows.Scan(&l.ID, &l.ShortCode, &l.OriginalURL, &l.Title, &l.Description,
			&l.ImageURL, &l.Domain, &l.ClickCount, &l.IsActive,
			&l.ExpiresAt, &l.CreatedAt, &l.UpdatedAt, &l.FolderID); err != nil {
			log.Printf("scan links failed: %v", err)
			continue
		}
		l.ShortURL = s.BuildShortURL(l.Domain, l.ShortCode)
		links = append(links, l)
	}

	return &domain.PaginatedLinks{
		Links: links, TotalCount: total, Page: page, PageSize: pageSize,
	}, nil
}

// Update 更新链接
func (s *LinkService) Update(ctx context.Context, linkID, userID int64, req domain.UpdateLinkRequest) (*domain.LinkInfo, error) {
	if req.OriginalURL != nil && *req.OriginalURL != "" && !urlcheck.IsSafeTarget(*req.OriginalURL) {
		return nil, errors.New("目标链接仅支持 http/https 协议")
	}

	// 校验文件夹归属（req 无 WorkspaceID 字段）
	if err := s.validateOwnedRefs(ctx, userID, req.FolderID, nil); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, errors.New("过期时间格式无效，请使用 RFC3339 格式（如 2025-01-01T00:00:00Z）")
		}
		expiresAt = &t
	}

	var passwordHash *string
	if req.Password != nil {
		hash := hashPassword(*req.Password)
		passwordHash = &hash
	}

	// 获取旧短码（用于缓存失效）；失败不影响主流程
	var oldShortCode string
	_ = s.db.QueryRow(ctx, `SELECT short_code FROM links WHERE id = $1`, linkID).Scan(&oldShortCode)

	// 校验自定义短码
	if req.ShortCode != nil && *req.ShortCode != "" {
		if !shortCodePattern.MatchString(*req.ShortCode) {
			return nil, errors.New("短码格式无效：只允许字母、数字、下划线和连字符，长度4-20位")
		}
		var exists bool
		// 快速路径检查：失败时不做判断，最终由唯一约束仲裁
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1 AND id != $2)`, *req.ShortCode, linkID).Scan(&exists); err == nil && exists {
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
		if _, err := s.db.Exec(ctx, `DELETE FROM link_tags WHERE link_id = $1`, linkID); err != nil {
			log.Printf("clear link tags failed: %v", err)
		}
		for _, tagID := range req.TagIDs {
			// 仅允许挂接自己的标签
			var tagOwner int64
			if err := s.db.QueryRow(ctx, `SELECT user_id FROM tags WHERE id = $1`, tagID).Scan(&tagOwner); err != nil || tagOwner != userID {
				log.Printf("skip non-owned tag %d for link %d (user %d)", tagID, linkID, userID)
				continue
			}
			if _, err := s.db.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, linkID, tagID); err != nil {
				log.Printf("attach tag %d to link %d failed: %v", tagID, linkID, err)
			}
		}
	}

	// 使缓存失效
	if s.cache != nil {
		if oldShortCode != "" {
			s.cache.InvalidateLink(ctx, oldShortCode)
		}
		if info.ShortCode != oldShortCode {
			s.cache.InvalidateLink(ctx, info.ShortCode)
		}
	}

	info.ShortURL = s.BuildShortURL(info.Domain, info.ShortCode)
	return &info, nil
}

// Delete 删除链接
func (s *LinkService) Delete(ctx context.Context, linkID, userID int64) error {
	// 获取短码用于缓存失效；失败不影响主流程
	var shortCode string
	_ = s.db.QueryRow(ctx, `SELECT short_code FROM links WHERE id = $1`, linkID).Scan(&shortCode)

	_, err := s.db.Exec(ctx, `DELETE FROM links WHERE id = $1 AND user_id = $2`, linkID, userID)
	if err != nil {
		return errors.New("删除链接失败")
	}

	if s.cache != nil && shortCode != "" {
		s.cache.InvalidateLink(ctx, shortCode)
	}
	return nil
}

// BatchDelete 批量删除链接
func (s *LinkService) BatchDelete(ctx context.Context, ids []int64, userID int64) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM links WHERE id = ANY($1) AND user_id = $2`, ids, userID)
	if err != nil {
		return 0, errors.New("批量删除失败")
	}
	return tag.RowsAffected(), nil
}

// BatchTag 批量打标签
func (s *LinkService) BatchTag(ctx context.Context, ids []int64, tagID int64, userID int64) error {
	// 标签必须属于当前用户
	var tagOwner int64
	if err := s.db.QueryRow(ctx, `SELECT user_id FROM tags WHERE id = $1`, tagID).Scan(&tagOwner); err != nil || tagOwner != userID {
		return errors.New("标签不存在或无权限")
	}
	for _, linkID := range ids {
		// 验证链接属于该用户
		var ownerID int64
		err := s.db.QueryRow(ctx, `SELECT user_id FROM links WHERE id = $1`, linkID).Scan(&ownerID)
		if err != nil || ownerID != userID {
			continue
		}
		if _, err := s.db.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, linkID, tagID); err != nil {
			log.Printf("batch tag link %d with tag %d failed: %v", linkID, tagID, err)
		}
	}
	return nil
}

// validateOwnedRefs 校验文件夹/工作区归属当前用户（nil 或 0 视为未设置）
func (s *LinkService) validateOwnedRefs(ctx context.Context, userID int64, folderID, workspaceID *int64) error {
	if folderID != nil && *folderID != 0 {
		var owner int64
		if err := s.db.QueryRow(ctx, `SELECT user_id FROM folders WHERE id = $1`, *folderID).Scan(&owner); err != nil || owner != userID {
			return errors.New("文件夹不存在或无权限")
		}
	}
	if workspaceID != nil && *workspaceID != 0 {
		var owner int64
		if err := s.db.QueryRow(ctx, `SELECT user_id FROM workspaces WHERE id = $1`, *workspaceID).Scan(&owner); err != nil || owner != userID {
			return errors.New("工作区不存在或无权限")
		}
	}
	return nil
}

// ExportCSV 导出用户链接为 CSV 字符串
func (s *LinkService) ExportCSV(ctx context.Context, userID int64) (string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT short_code, original_url, COALESCE(title,''), domain, click_count, is_active, created_at
		FROM links WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return "", errors.New("查询链接数据失败")
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("短码,目标URL,标题,域名,点击量,状态,创建时间\n")
	for rows.Next() {
		var code, url, title, domain string
		var clicks int64
		var active bool
		var created time.Time
		if err := rows.Scan(&code, &url, &title, &domain, &clicks, &active, &created); err != nil {
			log.Printf("export csv scan failed: %v", err)
			continue
		}
		status := "启用"
		if !active {
			status = "停用"
		}
		fmt.Fprintf(&sb, "%s,%s,%s,%s,%d,%s,%s\n",
			escapeCSV(code), escapeCSV(url), escapeCSV(title), escapeCSV(domain), clicks, status, created.Format("2006-01-02 15:04"))
	}
	return sb.String(), nil
}

// escapeCSV 转义 CSV 字段：以 = + - @ 或制表符开头的字段加单引号前缀
// （防止 Excel 公式注入），含逗号/引号/换行的字段用引号包裹。
func escapeCSV(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	if strings.HasPrefix(s, "=") || strings.HasPrefix(s, "+") ||
		strings.HasPrefix(s, "-") || strings.HasPrefix(s, "@") ||
		strings.HasPrefix(s, "\t") {
		s = "'" + s
	}
	if strings.ContainsAny(s, ",\"\n") {
		s = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// LogClick 发布点击事件到 Kafka；Kafka 不可用时回退直写，保证点击不丢
func (s *LinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
	if s.kafka != nil {
		if err := s.kafka.PublishClick(ctx, mq.ClickEvent{
			LinkID:    linkID,
			IP:        ip,
			UserAgent: userAgent,
			Platform:  platform,
			Referer:   referer,
			CreatedAt: time.Now(),
		}); err != nil {
			// Kafka 失败 → 落到直写
			log.Printf("kafka publish failed, falling back to direct write: %v", err)
		} else {
			return
		}
	}
	if s.clickWriter != nil {
		if err := s.clickWriter.WriteClick(ctx, linkID, ip, userAgent, platform, referer, time.Now()); err != nil {
			log.Printf("click direct write failed: %v", err)
		}
	}
}

// BuildShortURL 构建完整短链接
func (s *LinkService) BuildShortURL(domain, code string) string {
	return "https://" + domain + "/r/" + code
}

// generateShortCode 生成6字节随机短码（12位十六进制，48bit 熵，
// 约 1670 万条链接才达 50% 生日碰撞概率）
func generateShortCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// hashPassword 使用 bcrypt 哈希链接访问密码（bcrypt 上限 72 字节，超长自动截断）
func hashPassword(pwd string) string {
	if len(pwd) > 72 {
		pwd = pwd[:72]
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		// bcrypt 失败极罕见（内部错误），退化为可验证的错误值
		return ""
	}
	return string(hash)
}

// checkPasswordHash 校验链接访问密码。
// 新哈希为 bcrypt；兼容存量未加盐 SHA-256 十六进制哈希（64 位 hex）。
func checkPasswordHash(password, hash string) bool {
	if hash == "" {
		return false
	}
	if len(hash) == 64 && isHexString(hash) {
		// 存量 SHA-256 哈希，常数时间比较
		legacy := sha256Hex(password)
		return hmac.Equal([]byte(legacy), []byte(hash))
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func isHexString(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
