-- 000011_email_lower_unique.up.sql
-- 消除 email 大小写重复：对 LOWER(email) 建立唯一索引。
-- 应用层（internal/service/auth_service.go）已同步统一小写存储与查询，保持前后一致。
-- 例如 User@x.com 与 user@x.com 会被视为同一邮箱。
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower
    ON users (LOWER(email))
    WHERE email IS NOT NULL;
