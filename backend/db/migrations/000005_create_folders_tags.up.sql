-- 文件夹
CREATE TABLE IF NOT EXISTS folders (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_folders_user ON folders(user_id);

-- 标签
CREATE TABLE IF NOT EXISTS tags (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(50) NOT NULL,
    color       VARCHAR(7) DEFAULT '#6366F1',
    created_at  TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_tags_user ON tags(user_id);

-- 链接-标签关联
CREATE TABLE IF NOT EXISTS link_tags (
    link_id     BIGINT NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    tag_id      BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (link_id, tag_id)
);

-- 为 links 增加 folder_id 字段
ALTER TABLE links ADD COLUMN IF NOT EXISTS folder_id BIGINT REFERENCES folders(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_links_folder ON links(folder_id);
