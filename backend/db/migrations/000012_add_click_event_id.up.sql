-- 000012_add_click_event_id.up.sql
-- 为点击日志增加幂等键 event_id（Kafka at-least-once 重投时用于去重）。
-- 生产端为每条点击生成一个随机 id，落库时 ON CONFLICT (event_id) DO NOTHING，
-- 重复消息不再累加 click_count，避免重投导致计数偏大。
ALTER TABLE click_logs ADD COLUMN IF NOT EXISTS event_id VARCHAR(64);
CREATE UNIQUE INDEX IF NOT EXISTS idx_click_logs_event_id ON click_logs(event_id);
