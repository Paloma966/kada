-- 000002_create_links.up.sql
-- 短链接表

CREATE TABLE links (
    id              BIGSERIAL PRIMARY KEY,
    short_code      VARCHAR(20) UNIQUE NOT NULL,       -- 短码（如 "abc123"）
    original_url    TEXT NOT NULL,                     -- 原始长链接
    title           VARCHAR(500),                      -- 标题（OG metadata）
    description     TEXT,                              -- 描述
    image_url       TEXT,                              -- 预览图
    domain          VARCHAR(255) DEFAULT 'kada.link',  -- 域名
    password_hash   VARCHAR(255),                      -- 访问密码（可选）
    expires_at      TIMESTAMPTZ,                       -- 过期时间
    is_active       BOOLEAN DEFAULT TRUE,              -- 是否启用
    click_count     BIGINT DEFAULT 0,                  -- 点击计数
    user_id         BIGINT REFERENCES users(id) ON DELETE SET NULL,  -- 创建者
    workspace_id    BIGINT,                            -- 所属工作区（后续添加外键）
    utm_source      VARCHAR(255),                      -- UTM 参数
    utm_medium      VARCHAR(255),
    utm_campaign    VARCHAR(255),
    utm_term        VARCHAR(255),
    utm_content     VARCHAR(255),
    ios_url         TEXT,                              -- iOS deeplink
    android_url     TEXT,                              -- Android deeplink
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_links_short_code ON links(short_code);
CREATE INDEX idx_links_user_id ON links(user_id);
CREATE INDEX idx_links_domain_code ON links(domain, short_code);
CREATE INDEX idx_links_workspace_id ON links(workspace_id);
CREATE INDEX idx_links_created_at ON links(created_at DESC);
