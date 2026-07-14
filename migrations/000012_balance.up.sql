-- Pay-per-post balance system: users hold a balance in UZS (so'm),
-- topped up manually via cheque upload reviewed by an admin.

ALTER TABLE users ADD COLUMN balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0);

CREATE TABLE balance_topups (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount > 0),
    cheque_path TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    review_note TEXT,
    reviewed_by UUID REFERENCES admins(id),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_balance_topups_user ON balance_topups(user_id, created_at DESC);
CREATE INDEX idx_balance_topups_status ON balance_topups(status, created_at ASC);
