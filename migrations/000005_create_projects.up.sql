CREATE TABLE IF NOT EXISTS projects (
    id BIGSERIAL PRIMARY KEY,
    public_id UUID UNIQUE NOT NULL,
    manager_id BIGINT NOT NULL REFERENCES managers(user_id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    claim_amount BIGINT NOT NULL CHECK (claim_amount > 0),
    envelope_total BIGINT NOT NULL CHECK (envelope_total >= claim_amount),
    envelope_remaining BIGINT NOT NULL CHECK (envelope_remaining >= 0),
    status VARCHAR(50) NOT NULL CHECK (status IN ('open', 'closed')) DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_projects_public_id ON projects(public_id);
CREATE INDEX idx_projects_manager_id ON projects(manager_id);
