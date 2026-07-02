-- 000003_games.up.sql
-- Create games table with indexes and trigger

CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(120) NOT NULL UNIQUE,
    icon TEXT,
    genres TEXT[],
    posts_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_games_slug ON games(slug);
CREATE INDEX idx_games_active_name ON games(is_active, name);
CREATE INDEX idx_games_posts_count ON games(posts_count DESC);
CREATE INDEX idx_games_name_trgm ON games USING gin(name gin_trgm_ops);
CREATE INDEX idx_games_genres ON games USING gin(genres);

-- Update trigger
CREATE TRIGGER set_games_updated_at
    BEFORE UPDATE ON games
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
