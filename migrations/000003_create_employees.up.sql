CREATE TABLE IF NOT EXISTS employees (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    reimburse_available BIGINT NOT NULL DEFAULT 0 CHECK (reimburse_available >= 0),
    reimburse_locked BIGINT NOT NULL DEFAULT 0 CHECK (reimburse_locked >= 0)
);
