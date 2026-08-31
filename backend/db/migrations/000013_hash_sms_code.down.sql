-- 000013_hash_sms_code.down.sql
-- 哈希不可逆，无法还原原明文；回退为可空 code 列以保持结构可回滚。
ALTER TABLE sms_codes ADD COLUMN IF NOT EXISTS code VARCHAR(6);
DROP INDEX IF EXISTS idx_sms_codes_code_hash;
ALTER TABLE sms_codes DROP COLUMN IF EXISTS code_hash;
