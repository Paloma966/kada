CREATE TABLE IF NOT EXISTS utm_templates (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    utm_source   VARCHAR(255),
    utm_medium   VARCHAR(255),
    utm_campaign VARCHAR(255),
    utm_term     VARCHAR(255),
    utm_content  VARCHAR(255),
    created_at   TIMESTAMP DEFAULT NOW(),
    updated_at   TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_utm_templates_user_id ON utm_templates(user_id);
