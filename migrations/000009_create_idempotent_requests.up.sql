CREATE TABLE IF NOT EXISTS idempotent_requests (
    key VARCHAR(255) PRIMARY KEY,
    status_code INT NOT NULL,
    response_body JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
