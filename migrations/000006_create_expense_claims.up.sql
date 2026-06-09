CREATE TABLE IF NOT EXISTS expense_claims (
    id BIGSERIAL PRIMARY KEY,
    public_id UUID UNIQUE NOT NULL,
    project_id BIGINT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES employees(user_id) ON DELETE CASCADE,
    receipt_url TEXT NOT NULL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')) DEFAULT 'pending',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_project_employee_claim UNIQUE(project_id, employee_id)
);

CREATE INDEX idx_expense_claims_public_id ON expense_claims(public_id);
CREATE INDEX idx_expense_claims_employee_id ON expense_claims(employee_id);
