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

// shortCodePattern allows only letters, digits, underscores and hyphens in a short code
var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{4,20}$`)

type LinkService struct {
	db          *pgxpool.Pool
	baseURL     string
	cache       *CacheService
	kafka       mq.ClickPublisher // Kafka publisher; nil means disabled
	clickWriter ClickWriter       // direct write (used by the degraded fallback)
}

func NewLinkService(db *pgxpool.Pool, baseURL string, cache *CacheService, kafka mq.ClickPublisher, clickWriter ClickWriter) *LinkService {
	return &LinkService{db: db, baseURL: baseURL, cache: cache, kafka: kafka, clickWriter: clickWriter}
}

// Create creates a short link
func (s *LinkService) Create(ctx context.Context, userID int64, req domain.CreateLinkRequest) (*domain.LinkInfo, error) {
	if !urlcheck.IsSafeTarget(req.OriginalURL) {
		return nil, errors.New("target URL must use http or https")
	}

	// validate folder/workspace ownership (previously any ID could be attached, leaking names across users)
	if err := s.validateOwnedRefs(ctx, userID, req.FolderID, req.WorkspaceID); err != nil {
		return nil, err
	}

	var shortCode string

	if req.ShortCode != nil && *req.ShortCode != "" {
		if !shortCodePattern.MatchString(*req.ShortCode) {
			return nil, errors.New("invalid short code format: only letters, digits, underscores and hyphens are allowed, length 4-20")
		}
		var exists bool
		// fast-path check: on failure make no decision, the unique constraint is the final arbiter
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1)`, *req.ShortCode).Scan(&exists); err == nil && exists {
			return nil, errors.New("that short code is already taken, please choose another")
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
			return nil, errors.New("invalid expiry format, use RFC 3339 (for example 2025-01-01T00:00:00Z)")
		}
		expiresAt = &t
	}

	var passwordHash *string
	if req.Password != nil && *req.Password != "" {
		hash := hashPassword(*req.Password)
		passwordHash = &hash
	}

	var info domain.LinkInfo
	// the unique constraint is the final arbiter for short-code conflicts: under a check-then-insert race the INSERT reports 23505,
	// a custom short code returns a friendly error while a random short code is regenerated and retried
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
				return nil, errors.New("that short code is already taken, please choose another")
			}
			if attempt >= 4 {
				log.Printf("create link failed: %v", err)
				return nil, errors.New("failed to generate short code, please try again")
			}
			shortCode = generateShortCode()
			continue
		}
		log.Printf("create link failed: %v", err)
		return nil, errors.New("failed to create short link")
	}

	// attach tags (only own tags may be attached, preventing tag name/color leaks across users)
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

	// write to cache
	if s.cache != nil {
		s.cache.SetLink(ctx, &info)
	}

	return &info, nil
}

// GetByID gets a link by ID (with full info such as folder, tags and UTM)
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

	// look up the folder name
	if info.FolderID != nil {
		var folderName string
		folderErr := s.db.QueryRow(ctx, `SELECT name FROM folders WHERE id = $1`, *info.FolderID).Scan(&folderName)
		if folderErr == nil {
			info.FolderName = &folderName
		}
	}

	// look up tags
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

// GetByCode gets a link by short code (reads from cache first)
func (s *LinkService) GetByCode(ctx context.Context, shortCode string) (*domain.LinkInfo, error) {
	// try to read from cache
	if s.cache != nil {
		if info, ok := s.cache.GetLink(ctx, shortCode); ok {
			// check whether it has expired
			if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
				s.cache.InvalidateLink(ctx, shortCode)
				return nil, errors.New("link has expired")
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

	// check whether it has expired
	if info.ExpiresAt != nil && info.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("link has expired")
	}

	info.ShortURL = s.BuildShortURL(info.Domain, info.ShortCode)

	// write to cache
	if s.cache != nil {
		s.cache.SetLink(ctx, &info)
	}

	return &info, nil
}

// HasPassword checks whether the link has a password set
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

// CheckPassword checks the link password
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
		return false, nil, errors.New("link has expired")
	}

	if passwordHash == nil || *passwordHash == "" {
		return true, &info, nil // no password
	}

	if password == "" || !checkPasswordHash(password, *passwordHash) {
		return false, &info, nil
	}

	info.ShortURL = s.BuildShortURL(info.Domain, info.ShortCode)
	return true, &info, nil
}

// List gets the user's link list
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
		return nil, errors.New("failed to list links")
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

// Update updates a link
func (s *LinkService) Update(ctx context.Context, linkID, userID int64, req domain.UpdateLinkRequest) (*domain.LinkInfo, error) {
	if req.OriginalURL != nil && *req.OriginalURL != "" && !urlcheck.IsSafeTarget(*req.OriginalURL) {
		return nil, errors.New("target URL must use http or https")
	}

	// validate folder ownership (req has no WorkspaceID field)
	if err := s.validateOwnedRefs(ctx, userID, req.FolderID, nil); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, errors.New("invalid expiry format, use RFC 3339 (for example 2025-01-01T00:00:00Z)")
		}
		expiresAt = &t
	}

	var passwordHash *string
	if req.Password != nil {
		hash := hashPassword(*req.Password)
		passwordHash = &hash
	}

	// get the old short code (used for cache invalidation); failure does not affect the main flow
	var oldShortCode string
	_ = s.db.QueryRow(ctx, `SELECT short_code FROM links WHERE id = $1`, linkID).Scan(&oldShortCode)

	// validate the custom short code
	if req.ShortCode != nil && *req.ShortCode != "" {
		if !shortCodePattern.MatchString(*req.ShortCode) {
			return nil, errors.New("invalid short code format: only letters, digits, underscores and hyphens are allowed, length 4-20")
		}
		var exists bool
		// fast-path check: on failure make no decision, the unique constraint is the final arbiter
		if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1 AND id != $2)`, *req.ShortCode, linkID).Scan(&exists); err == nil && exists {
			return nil, errors.New("that short code is already taken, please choose another")
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
		return nil, errors.New("failed to update link, link not found or access denied")
	}

	// update tag associations
	if req.TagIDs != nil {
		if _, err := s.db.Exec(ctx, `DELETE FROM link_tags WHERE link_id = $1`, linkID); err != nil {
			log.Printf("clear link tags failed: %v", err)
		}
		for _, tagID := range req.TagIDs {
			// only own tags may be attached
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

	// invalidate the cache
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

// Delete deletes a link
func (s *LinkService) Delete(ctx context.Context, linkID, userID int64) error {
	// get the short code for cache invalidation; failure does not affect the main flow
	var shortCode string
	_ = s.db.QueryRow(ctx, `SELECT short_code FROM links WHERE id = $1`, linkID).Scan(&shortCode)

	_, err := s.db.Exec(ctx, `DELETE FROM links WHERE id = $1 AND user_id = $2`, linkID, userID)
	if err != nil {
		return errors.New("failed to delete link")
	}

	if s.cache != nil && shortCode != "" {
		s.cache.InvalidateLink(ctx, shortCode)
	}
	return nil
}

// BatchDelete deletes links in bulk
func (s *LinkService) BatchDelete(ctx context.Context, ids []int64, userID int64) (int64, error) {
	// first fetch the short codes of the links to be deleted, to invalidate the cache afterwards (aligned with single Delete)
	var codes []string
	rows, err := s.db.Query(ctx, `SELECT short_code FROM links WHERE id = ANY($1) AND user_id = $2`, ids, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c string
			if scanErr := rows.Scan(&c); scanErr != nil {
				continue
			}
			codes = append(codes, c)
		}
	}

	tag, err := s.db.Exec(ctx, `DELETE FROM links WHERE id = ANY($1) AND user_id = $2`, ids, userID)
	if err != nil {
		return 0, errors.New("bulk delete failed")
	}

	// invalidate the cache of deleted short codes: otherwise a deleted link can still be redirected within the cache TTL
	if s.cache != nil {
		for _, c := range codes {
			s.cache.InvalidateLink(ctx, c)
		}
	}
	return tag.RowsAffected(), nil
}

// BatchTag tags links in bulk
func (s *LinkService) BatchTag(ctx context.Context, ids []int64, tagID int64, userID int64) error {
	// the tag must belong to the current user
	var tagOwner int64
	if err := s.db.QueryRow(ctx, `SELECT user_id FROM tags WHERE id = $1`, tagID).Scan(&tagOwner); err != nil || tagOwner != userID {
		return errors.New("tag not found or access denied")
	}
	for _, linkID := range ids {
		// verify the link belongs to that user
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

// validateOwnedRefs validates that the folder/workspace belongs to the current user (nil or 0 means unset)
func (s *LinkService) validateOwnedRefs(ctx context.Context, userID int64, folderID, workspaceID *int64) error {
	if folderID != nil && *folderID != 0 {
		var owner int64
		if err := s.db.QueryRow(ctx, `SELECT user_id FROM folders WHERE id = $1`, *folderID).Scan(&owner); err != nil || owner != userID {
			return errors.New("folder not found or access denied")
		}
	}
	if workspaceID != nil && *workspaceID != 0 {
		var owner int64
		if err := s.db.QueryRow(ctx, `SELECT user_id FROM workspaces WHERE id = $1`, *workspaceID).Scan(&owner); err != nil || owner != userID {
			return errors.New("workspace not found or access denied")
		}
	}
	return nil
}

// ExportCSV exports the user's links as a CSV string
func (s *LinkService) ExportCSV(ctx context.Context, userID int64) (string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT short_code, original_url, COALESCE(title,''), domain, click_count, is_active, created_at
		FROM links WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return "", errors.New("failed to query link data")
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("code,target_url,title,domain,clicks,status,created_at\n")
	for rows.Next() {
		var code, url, title, domain string
		var clicks int64
		var active bool
		var created time.Time
		if err := rows.Scan(&code, &url, &title, &domain, &clicks, &active, &created); err != nil {
			log.Printf("export csv scan failed: %v", err)
			continue
		}
		status := "enabled"
		if !active {
			status = "disabled"
		}
		fmt.Fprintf(&sb, "%s,%s,%s,%s,%d,%s,%s\n",
			escapeCSV(code), escapeCSV(url), escapeCSV(title), escapeCSV(domain), clicks, status, created.Format("2006-01-02 15:04"))
	}
	return sb.String(), nil
}

// escapeCSV escapes a CSV field: fields starting with = + - @ or a tab get a single-quote prefix
// (preventing Excel formula injection), and fields containing commas/quotes/newlines are wrapped in quotes.
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

// LogClick publishes a click event to Kafka; falls back to a direct write when Kafka is unavailable so clicks are not lost
func (s *LinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
	// generate an idempotency key: used to deduplicate on Kafka redelivery or degraded direct write
	eventID := newEventID()
	if s.kafka != nil {
		if err := s.kafka.PublishClick(ctx, mq.ClickEvent{
			EventID:   eventID,
			LinkID:    linkID,
			IP:        ip,
			UserAgent: userAgent,
			Platform:  platform,
			Referer:   referer,
			CreatedAt: time.Now(),
		}); err != nil {
			// Kafka failed -> fall back to direct write
			log.Printf("kafka publish failed, falling back to direct write: %v", err)
		} else {
			return
		}
	}
	if s.clickWriter != nil {
		if err := s.clickWriter.WriteClick(ctx, eventID, linkID, ip, userAgent, platform, referer, time.Now()); err != nil {
			log.Printf("click direct write failed: %v", err)
		}
	}
}

// BuildShortURL builds the full short URL
func (s *LinkService) BuildShortURL(domain, code string) string {
	return "https://" + domain + "/r/" + code
}

// newEventID generates the click event idempotency key (16 random bytes -> 32 hex chars).
// used to deduplicate Kafka at-least-once redeliveries / degraded direct writes, avoiding double increments of click_count.
func newEventID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateShortCode generates a 6-byte random short code (12 hex digits, 48 bits of entropy,
// about 16.7 million links to reach a 50% birthday collision probability)
func generateShortCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// hashPassword hashes the link access password with bcrypt (bcrypt caps at 72 bytes, longer input is truncated automatically)
func hashPassword(pwd string) string {
	if len(pwd) > 72 {
		pwd = pwd[:72]
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		// bcrypt failure is extremely rare (internal error), degrade to a value that never verifies
		return ""
	}
	return string(hash)
}

// checkPasswordHash verifies the link access password.
// new hashes are bcrypt; legacy unsalted SHA-256 hex hashes (64 hex digits) are still supported.
func checkPasswordHash(password, hash string) bool {
	if hash == "" {
		return false
	}
	if len(hash) == 64 && isHexString(hash) {
		// legacy SHA-256 hash, constant-time comparison
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
