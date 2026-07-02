-- 000009_reports.down.sql
-- Drop reports table and enums

DROP TRIGGER IF EXISTS set_reports_updated_at ON reports;
DROP TABLE IF EXISTS reports;

DROP TYPE IF EXISTS report_action;
DROP TYPE IF EXISTS report_priority;
DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS report_category;
DROP TYPE IF EXISTS report_type;
