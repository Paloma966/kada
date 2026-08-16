-- 000010_add_sms_code_attempts.up.sql
-- 验证码尝试次数：限制单个验证码可被错误尝试的次数（防爆破）
ALTER TABLE sms_codes ADD COLUMN IF NOT EXISTS attempts INT NOT NULL DEFAULT 0;
