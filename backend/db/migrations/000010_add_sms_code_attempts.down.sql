-- 000010_add_sms_code_attempts.down.sql
ALTER TABLE sms_codes DROP COLUMN IF EXISTS attempts;
