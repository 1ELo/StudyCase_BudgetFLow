CREATE TABLE IF NOT EXISTS managers (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    budget_available BIGINT NOT NULL DEFAULT 0 CHECK (budget_available >= 0),
    budget_locked BIGINT NOT NULL DEFAULT 0 CHECK (budget_locked >= 0)
);
