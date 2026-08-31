-- 000012_add_click_event_id.down.sql
DROP INDEX IF EXISTS idx_click_logs_event_id;
ALTER TABLE click_logs DROP COLUMN IF EXISTS event_id;
