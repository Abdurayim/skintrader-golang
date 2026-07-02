-- 000009_reports.up.sql
-- Create report-related enums, table, indexes, and trigger

-- Enums
CREATE TYPE report_type AS ENUM ('post', 'user');
CREATE TYPE report_category AS ENUM (
    'scam',
    'fake_item',
    'inappropriate_content',
    'duplicate_post',
    'incorrect_pricing',
    'harassment',
    'spam',
    'fraud',
    'impersonation',
    'offensive_profile',
    'other'
);
CREATE TYPE report_status AS ENUM ('pending', 'under_review', 'resolved', 'dismissed');
CREATE TYPE report_priority AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE report_action AS ENUM ('dismiss', 'delete_post', 'warn_user', 'suspend_user', 'ban_user', 'delete_user');

-- Reports table
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    report_type report_type NOT NULL,
    target_id UUID NOT NULL,
    target_model VARCHAR(10) NOT NULL CHECK (target_model IN ('Post', 'User')),
    category report_category NOT NULL,
    description VARCHAR(1000),
    status report_status NOT NULL DEFAULT 'pending',
    priority report_priority NOT NULL DEFAULT 'low',
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    resolution_action report_action,
    resolution_notes VARCHAR(2000),
    resolution_admin_notes VARCHAR(2000),
    resolved_at TIMESTAMPTZ,
    report_hash VARCHAR(32) UNIQUE,
    ip_address INET,
    user_agent TEXT,
    report_count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_reports_reporter_created ON reports(reporter_id, created_at);
CREATE INDEX idx_reports_target ON reports(target_id, report_type);
CREATE INDEX idx_reports_status_priority_created ON reports(status, priority, created_at);
CREATE INDEX idx_reports_category_status ON reports(category, status);
CREATE INDEX idx_reports_hash ON reports(report_hash);

-- Update trigger
CREATE TRIGGER set_reports_updated_at
    BEFORE UPDATE ON reports
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
