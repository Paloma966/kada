-- 000003_create_click_logs.up.sql
-- 点击日志表（含平台检测信息）

CREATE TYPE click_platform AS ENUM ('browser', 'wechat', 'qq', 'weibo', 'xiaohongshu', 'sms', 'unknown');

CREATE TABLE click_logs (
    id              BIGSERIAL PRIMARY KEY,
    link_id         BIGINT REFERENCES links(id) ON DELETE CASCADE,
    ip              VARCHAR(45),                       -- 访问者 IP
    user_agent      TEXT,                              -- User-Agent 字符串
    platform        click_platform DEFAULT 'unknown',   -- 平台类型
    referer         TEXT,                              -- Referer
    country         VARCHAR(10),                       -- 国家代码
    province        VARCHAR(50),                       -- 省份
    city            VARCHAR(50),                       -- 城市
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_click_logs_link_id ON click_logs(link_id);
CREATE INDEX idx_click_logs_created_at ON click_logs(created_at);
CREATE INDEX idx_click_logs_platform ON click_logs(platform);
