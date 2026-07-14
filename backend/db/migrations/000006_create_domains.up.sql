CREATE TABLE IF NOT EXISTS domains (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    verified    BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP,
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_domains_user_id ON domains(user_id);
