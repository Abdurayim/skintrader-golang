-- Thumbnail files were never generated, but post_images rows recorded a
-- _thumb path anyway, so the frontend requested images that 404.
-- Clear them; the frontend falls back to original_path when the
-- thumbnail is absent.
UPDATE post_images SET thumbnail_path = NULL WHERE thumbnail_path IS NOT NULL;
