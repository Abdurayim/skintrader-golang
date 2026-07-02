-- 000005_subscriptions.up.sql
-- Create subscriptions table with indexes and trigger

-- Enum
CREATE TYPE subscription_plan AS ENUM ('monthly');

-- Subscriptions table
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status subscription_status NOT NULL DEFAULT 'pending',
    plan subscription_plan NOT NULL,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    auto_renew BOOLEAN NOT NULL DEFAULT TRUE,
    last_payment_id UUID,
    grace_period_started TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_subscriptions_user_status ON subscriptions(user_id, status);
CREATE INDEX idx_subscriptions_status_end_date ON subscriptions(status, end_date);
CREATE INDEX idx_subscriptions_created_at ON subscriptions(created_at DESC);

-- Update trigger
CREATE TRIGGER set_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
