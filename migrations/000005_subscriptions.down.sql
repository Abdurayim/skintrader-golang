-- 000005_subscriptions.down.sql
-- Drop subscriptions table and enum

DROP TRIGGER IF EXISTS set_subscriptions_updated_at ON subscriptions;
DROP TABLE IF EXISTS subscriptions;

DROP TYPE IF EXISTS subscription_plan;
