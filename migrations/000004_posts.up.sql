-- 000004_posts.up.sql
-- Create post-related enums, tables, indexes, and triggers

-- Enums
CREATE TYPE post_status AS ENUM ('active', 'sold');
CREATE TYPE post_type AS ENUM ('skin', 'profile');
CREATE TYPE currency_code AS ENUM ('UZS', 'USD');

-- Posts table
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(100) NOT NULL,
    description VARCHAR(2000),
    price NUMERIC(12, 2) NOT NULL CHECK (price >= 0),
    currency currency_code NOT NULL DEFAULT 'UZS',
    game_id UUID NOT NULL REFERENCES games(id),
    genre VARCHAR(50),
    type post_type NOT NULL,
    contact_info JSONB,
    status post_status NOT NULL DEFAULT 'active',
    views_count INTEGER NOT NULL DEFAULT 0,
    reports_count INTEGER NOT NULL DEFAULT 0,
    reported_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID,
    deleted_by_type VARCHAR(10) CHECK (deleted_by_type IN ('User', 'Admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Post Images table
CREATE TABLE post_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    original_path TEXT NOT NULL,
    thumbnail_path TEXT,
    filename VARCHAR(255) NOT NULL,
    size INTEGER NOT NULL,
    mime_type VARCHAR(50) NOT NULL,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes on posts
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_posts_game_id ON posts(game_id);
CREATE INDEX idx_posts_status_created_at ON posts(status, created_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_game_status_created_at ON posts(game_id, status, created_at);
CREATE INDEX idx_posts_type_status_created_at ON posts(type, status, created_at);
CREATE INDEX idx_posts_price_currency ON posts(price, currency);
CREATE INDEX idx_posts_genre ON posts(genre);
CREATE INDEX idx_posts_title_trgm ON posts USING gin(title gin_trgm_ops);
CREATE INDEX idx_posts_description_trgm ON posts USING gin(description gin_trgm_ops);
CREATE INDEX idx_posts_fts ON posts USING gin(to_tsvector('simple', title || ' ' || COALESCE(description, '')));

-- Indexes on post_images
CREATE INDEX idx_post_images_post_id_sort ON post_images(post_id, sort_order);

-- Update trigger
CREATE TRIGGER set_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
