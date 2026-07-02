-- 000002_users.up.sql
-- Create user-related enums, tables, indexes, and triggers

-- Enums
CREATE TYPE user_status AS ENUM ('active', 'suspended', 'banned');
CREATE TYPE kyc_status AS ENUM ('not_submitted', 'pending', 'verified', 'rejected');
CREATE TYPE language_code AS ENUM ('en', 'ru', 'uz');
CREATE TYPE subscription_status AS ENUM ('none', 'active', 'expired', 'grace_period', 'pending', 'cancelled');
CREATE TYPE auth_provider AS ENUM ('google', 'apple', 'email');

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_provider auth_provider NOT NULL,
    google_id VARCHAR(255) UNIQUE,
    apple_id VARCHAR(255) UNIQUE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    display_name VARCHAR(50),
    phone_number VARCHAR(20),
    bio VARCHAR(500),
    avatar_url TEXT,
    social_media JSONB,
    language language_code NOT NULL DEFAULT 'en',
    status user_status NOT NULL DEFAULT 'active',
    status_reason TEXT,
    kyc_status kyc_status NOT NULL DEFAULT 'not_submitted',
    kyc_rejection_reason TEXT,
    kyc_verified_at TIMESTAMPTZ,
    kyc_reviewed_by UUID,
    face_match_score REAL,
    location GEOGRAPHY(Point, 4326),
    location_updated_at TIMESTAMPTZ,
    posts_count INTEGER NOT NULL DEFAULT 0,
    reports_received INTEGER NOT NULL DEFAULT 0,
    reports_made INTEGER NOT NULL DEFAULT 0,
    subscription_status subscription_status NOT NULL DEFAULT 'none',
    current_subscription_id UUID,
    subscription_expires_at TIMESTAMPTZ,
    grace_period_ends_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    last_active_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- KYC Documents table
CREATE TABLE kyc_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    doc_type VARCHAR(20) NOT NULL CHECK (doc_type IN ('id_card', 'passport', 'selfie')),
    file_path TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified_at TIMESTAMPTZ,
    UNIQUE (user_id, doc_type)
);

-- Refresh Tokens table
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    device_info TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes on users
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_apple_id ON users(apple_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_kyc_status ON users(kyc_status);
CREATE INDEX idx_users_subscription_status ON users(subscription_status);
CREATE INDEX idx_users_created_at ON users(created_at DESC);
CREATE INDEX idx_users_display_name_trgm ON users USING gin(display_name gin_trgm_ops);
CREATE INDEX idx_users_location ON users USING gist(location);

-- Update trigger
CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
