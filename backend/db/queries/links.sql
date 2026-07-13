-- name: CreateLink :one
INSERT INTO links (short_code, original_url, title, description, image_url, domain, password_hash, expires_at, user_id, workspace_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetLinkByCode :one
SELECT * FROM links WHERE short_code = $1 AND is_active = TRUE;

-- name: GetLinkByDomainAndCode :one
SELECT * FROM links WHERE domain = $1 AND short_code = $2 AND is_active = TRUE;

-- name: GetLinksByUser :many
SELECT * FROM links WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetLinksByWorkspace :many
SELECT * FROM links WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateLink :one
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
RETURNING *;

-- name: DeleteLink :exec
DELETE FROM links WHERE id = $1 AND user_id = $2;

-- name: ArchiveLink :exec
UPDATE links SET is_active = FALSE, updated_at = NOW() WHERE id = $1 AND user_id = $2;

-- name: IncrementClickCount :exec
UPDATE links SET click_count = click_count + 1 WHERE id = $1;

-- name: LogClick :one
INSERT INTO click_logs (link_id, ip, user_agent, platform, referer)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at;
