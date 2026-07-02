-- 000004_posts.down.sql
-- Drop post-related tables and enums

DROP TRIGGER IF EXISTS set_posts_updated_at ON posts;

DROP TABLE IF EXISTS post_images;
DROP TABLE IF EXISTS posts;

DROP TYPE IF EXISTS currency_code;
DROP TYPE IF EXISTS post_type;
DROP TYPE IF EXISTS post_status;
