-- 000003_games.down.sql
-- Drop games table

DROP TRIGGER IF EXISTS set_games_updated_at ON games;
DROP TABLE IF EXISTS games;
