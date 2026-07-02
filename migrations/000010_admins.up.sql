-- 000010_admins.up.sql
-- Create admin-related enum, tables, indexes, and trigger

-- Enum
CREATE TYPE admin_role AS ENUM ('superadmin', 'moderator', 'support');

-- Admins table
CREATE TABLE admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(50) NOT NULL,
    role admin_role NOT NULL DEFAULT 'moderator',
    permissions TEXT[],
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    password_reset_token VARCHAR(255),
    password_reset_expires TIMESTAMPTZ,
    created_by UUID REFERENCES admins(id),
    updated_by UUID REFERENCES admins(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Admin Refresh Tokens table
CREATE TABLE admin_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    device_info TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Admin Logs table
CREATE TABLE admin_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID REFERENCES admins(id),
    action VARCHAR(50) NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('User', 'Post', 'Report', 'Admin', 'Game', 'Subscription', 'Transaction', 'Conversation', 'Message')),
    target_id UUID,
    details JSONB,
    previous_state JSONB,
    new_state JSONB,
    ip_address INET,
    user_agent TEXT,
    reason TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes on admins
CREATE INDEX idx_admins_email ON admins(email);
CREATE INDEX idx_admins_role ON admins(role);
CREATE INDEX idx_admins_is_active ON admins(is_active);

-- Indexes on admin_refresh_tokens
CREATE INDEX idx_admin_refresh_tokens_admin_id ON admin_refresh_tokens(admin_id);
CREATE INDEX idx_admin_refresh_tokens_expires_at ON admin_refresh_tokens(expires_at);

-- Indexes on admin_logs
CREATE INDEX idx_admin_logs_admin_id ON admin_logs(admin_id);
CREATE INDEX idx_admin_logs_action ON admin_logs(action);
CREATE INDEX idx_admin_logs_target ON admin_logs(target_type, target_id);
CREATE INDEX idx_admin_logs_created_at ON admin_logs(created_at DESC);

-- Update trigger for admins
CREATE TRIGGER set_admins_updated_at
    BEFORE UPDATE ON admins
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
