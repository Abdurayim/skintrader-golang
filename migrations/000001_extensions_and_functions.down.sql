-- 000001_extensions_and_functions.down.sql
-- Drop utility functions and extensions

DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;
DROP EXTENSION IF EXISTS "postgis" CASCADE;
DROP EXTENSION IF EXISTS "pg_trgm" CASCADE;
DROP EXTENSION IF EXISTS "pgcrypto" CASCADE;
