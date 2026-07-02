-- 000010_admins.down.sql
-- Drop admin-related tables and enum

DROP TRIGGER IF EXISTS set_admins_updated_at ON admins;

DROP TABLE IF EXISTS admin_logs;
DROP TABLE IF EXISTS admin_refresh_tokens;
DROP TABLE IF EXISTS admins;

DROP TYPE IF EXISTS admin_role;
