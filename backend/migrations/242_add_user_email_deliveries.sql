CREATE TABLE IF NOT EXISTS user_email_deliveries (
    id BIGSERIAL PRIMARY KEY,
    event VARCHAR(100) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event, user_id),
    CONSTRAINT user_email_deliveries_status_check CHECK (status IN ('sent', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_user_email_deliveries_event_status
    ON user_email_deliveries (event, status);

CREATE INDEX IF NOT EXISTS idx_user_email_deliveries_sent_at
    ON user_email_deliveries (sent_at);
