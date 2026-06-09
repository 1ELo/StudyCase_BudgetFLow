CREATE TABLE IF NOT EXISTS budget_topups (
    id BIGSERIAL PRIMARY KEY,
    public_id UUID UNIQUE NOT NULL,
    manager_id BIGINT NOT NULL REFERENCES managers(user_id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')) DEFAULT 'pending',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_budget_topups_public_id ON budget_topups(public_id);
CREATE INDEX idx_budget_topups_manager_id ON budget_topups(manager_id);
