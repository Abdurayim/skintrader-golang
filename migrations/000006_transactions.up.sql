-- 000006_transactions.up.sql
-- Create transaction-related enums, table, indexes, trigger, and FK back to subscriptions

-- Enums
CREATE TYPE transaction_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled', 'refunded');
CREATE TYPE payment_method AS ENUM ('payme', 'click', 'xazna', 'uzum');

-- Transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id),
    external_transaction_id VARCHAR(255) UNIQUE,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    currency currency_code NOT NULL DEFAULT 'UZS',
    status transaction_status NOT NULL DEFAULT 'pending',
    payment_method payment_method NOT NULL,
    payment_response JSONB,
    webhook_received BOOLEAN NOT NULL DEFAULT FALSE,
    webhook_received_at TIMESTAMPTZ,
    ip_address INET,
    user_agent TEXT,
    error_message TEXT,
    error_code VARCHAR(100),
    refunded_at TIMESTAMPTZ,
    refund_reason TEXT,
    refunded_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add FK from subscriptions.last_payment_id to transactions(id)
ALTER TABLE subscriptions
    ADD CONSTRAINT fk_subscriptions_last_payment
    FOREIGN KEY (last_payment_id) REFERENCES transactions(id);

-- Indexes
CREATE INDEX idx_transactions_user_created ON transactions(user_id, created_at);
CREATE INDEX idx_transactions_status_created ON transactions(status, created_at);
CREATE INDEX idx_transactions_subscription_id ON transactions(subscription_id);
CREATE INDEX idx_transactions_external_id ON transactions(external_transaction_id);

-- Update trigger
CREATE TRIGGER set_transactions_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
