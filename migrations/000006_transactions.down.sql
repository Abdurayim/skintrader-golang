-- 000006_transactions.down.sql
-- Drop FK constraint on subscriptions, transactions table, and enums

DROP TRIGGER IF EXISTS set_transactions_updated_at ON transactions;

ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS fk_subscriptions_last_payment;

DROP TABLE IF EXISTS transactions;

DROP TYPE IF EXISTS payment_method;
DROP TYPE IF EXISTS transaction_status;
