-- name: CreateUser :one
INSERT INTO users (phone, email, name, password_hash, last_login_ip)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, phone, email, name, avatar, created_at, updated_at;

-- name: GetUserByPhone :one
SELECT id, phone, email, name, avatar, password_hash, wechat_openid, created_at, updated_at
FROM users WHERE phone = $1;

-- name: GetUserByEmail :one
SELECT id, phone, email, name, avatar, password_hash, wechat_openid, created_at, updated_at
FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT id, phone, email, name, avatar, wechat_openid, created_at, updated_at
FROM users WHERE id = $1;

-- name: GetUserByWechatOpenID :one
SELECT id, phone, email, name, avatar, wechat_openid, created_at, updated_at
FROM users WHERE wechat_openid = $1;

-- name: UpdateUserWechat :exec
UPDATE users SET wechat_openid = $2, name = COALESCE($3, name), avatar = COALESCE($4, avatar), updated_at = NOW()
WHERE id = $1;

-- name: UpdateLastLogin :exec
UPDATE users SET last_login_at = NOW(), last_login_ip = $2 WHERE id = $1;
