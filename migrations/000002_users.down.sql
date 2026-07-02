-- 000002_users.down.sql
-- Drop user-related tables and enums

DROP TRIGGER IF EXISTS set_users_updated_at ON users;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS kyc_documents;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS auth_provider;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS language_code;
DROP TYPE IF EXISTS kyc_status;
DROP TYPE IF EXISTS user_status;
