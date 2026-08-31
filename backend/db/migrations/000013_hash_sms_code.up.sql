-- 000013_hash_sms_code.up.sql
-- 验证码不落明文：新增 code_hash 列并回填已有数据，随后删除明文 code 列。
-- PostgreSQL 11+ 内置 sha256(bytea)，无需 pgcrypto。
ALTER TABLE sms_codes ADD COLUMN IF NOT EXISTS code_hash VARCHAR(64);

-- 回填既有明文为哈希（sha256 不可逆，仅迁移用）
UPDATE sms_codes SET code_hash = encode(sha256(code::bytea), 'hex')
  WHERE code_hash IS NULL AND code IS NOT NULL;

-- 删除明文列
ALTER TABLE sms_codes DROP COLUMN IF EXISTS code;

CREATE INDEX IF NOT EXISTS idx_sms_codes_code_hash ON sms_codes(code_hash);
