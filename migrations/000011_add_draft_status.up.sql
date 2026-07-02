-- 000011_add_draft_status.up.sql
-- Add 'draft' value to the post_status enum

ALTER TYPE post_status ADD VALUE IF NOT EXISTS 'draft';
