-- 000001_create_users.up.sql
-- 用户表：支持手机号、邮箱、微信多种登录方式

CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    phone           VARCHAR(20) UNIQUE,                -- 手机号（中国大陆 +86）
    email           VARCHAR(255) UNIQUE,               -- 邮箱
    wechat_openid   VARCHAR(128) UNIQUE,               -- 微信 OpenID
    wechat_unionid  VARCHAR(128),                      -- 微信 UnionID
    name            VARCHAR(100),                      -- 昵称
    avatar          TEXT,                              -- 头像 URL
    password_hash   VARCHAR(255),                      -- bcrypt 密码哈希
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ,                       -- 最后登录时间
    last_login_ip   VARCHAR(45)                        -- 最后登录 IP
);

CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_wechat_openid ON users(wechat_openid);
