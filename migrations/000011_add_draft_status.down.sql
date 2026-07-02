-- 000011_add_draft_status.down.sql
-- Note: PostgreSQL does not support removing enum values directly.
-- To rollback, you would need to recreate the enum type.
-- This is intentionally left as a no-op to avoid data loss.

-- UPDATE posts SET status = 'active' WHERE status = 'draft';
