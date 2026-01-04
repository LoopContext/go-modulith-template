-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

-- Add profile fields to users table
ALTER TABLE auth.users ADD COLUMN display_name VARCHAR(255);
ALTER TABLE auth.users ADD COLUMN avatar_url VARCHAR(512);

-- Sessions table for tracking user sessions and token management
CREATE TABLE auth.sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent VARCHAR(512),
    ip_address VARCHAR(45), -- IPv6 max length
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_active_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX idx_sessions_refresh_token_hash ON auth.sessions(refresh_token_hash);
CREATE INDEX idx_sessions_expires_at ON auth.sessions(expires_at);

-- Token blacklist for revoked access tokens
-- Tokens are stored until they naturally expire, then can be cleaned up
CREATE TABLE auth.token_blacklist (
    token_hash VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reason VARCHAR(50), -- 'logout', 'password_change', 'admin_revoke', etc.
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_token_blacklist_expires_at ON auth.token_blacklist(expires_at);

-- Pending email/phone changes (verification required)
CREATE TABLE auth.pending_contact_changes (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    change_type VARCHAR(10) NOT NULL, -- 'email' or 'phone'
    new_value VARCHAR(255) NOT NULL,
    verification_code VARCHAR(10) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_pending_contact_changes_user_id ON auth.pending_contact_changes(user_id);
CREATE INDEX idx_pending_contact_changes_expires_at ON auth.pending_contact_changes(expires_at);

-- Reset search_path to default
SET search_path TO public;

