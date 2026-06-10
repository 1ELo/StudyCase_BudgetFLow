CREATE TABLE IF NOT EXISTS payouts (
    id BIGSERIAL PRIMARY KEY,
    public_id UUID UNIQUE NOT NULL,
    employee_id BIGINT NOT NULL REFERENCES employees(user_id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount > 0),
    fee BIGINT CHECK (fee >= 0),
    net_amount BIGINT CHECK (net_amount >= 0),
    status VARCHAR(50) NOT NULL CHECK (status IN ('pending', 'completed', 'failed')) DEFAULT 'pending',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payouts_public_id ON payouts(public_id);
CREATE INDEX idx_payouts_employee_id ON payouts(employee_id);
