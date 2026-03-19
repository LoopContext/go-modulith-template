-- Create schema for auth module
CREATE SCHEMA IF NOT EXISTS auth;

-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

-- Function for automatic updated_at
CREATE OR REPLACE FUNCTION auth.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TABLE auth.users (
    id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50) UNIQUE,
    display_name VARCHAR(255),
    avatar_url VARCHAR(512),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    phone_verified BOOLEAN NOT NULL DEFAULT FALSE,
    timezone VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_status ON auth.users(status);

CREATE TABLE auth.roles (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth.permissions (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth.role_permissions (
    role_id VARCHAR(64) NOT NULL,
    permission_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES auth.roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES auth.permissions(id) ON DELETE CASCADE
);

CREATE TABLE auth.user_roles (
    user_id VARCHAR(64) NOT NULL,
    role_id VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES auth.roles(id) ON DELETE CASCADE
);

CREATE TABLE auth.magic_codes (
    code VARCHAR(10) NOT NULL,
    user_email VARCHAR(255),
    user_phone VARCHAR(50),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_magic_codes_email ON auth.magic_codes(user_email);
CREATE INDEX idx_magic_codes_phone ON auth.magic_codes(user_phone);

-- Sessions table
CREATE TABLE auth.sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent VARCHAR(512),
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX idx_sessions_refresh_token_hash ON auth.sessions(refresh_token_hash);
CREATE INDEX idx_sessions_expires_at ON auth.sessions(expires_at);

-- Token blacklist
CREATE TABLE auth.token_blacklist (
    token_hash VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason VARCHAR(50),
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_token_blacklist_expires_at ON auth.token_blacklist(expires_at);

-- Pending contact changes
CREATE TABLE auth.pending_contact_changes (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    change_type VARCHAR(10) NOT NULL,
    new_value VARCHAR(255) NOT NULL,
    verification_code VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX idx_pending_contact_changes_user_id ON auth.pending_contact_changes(user_id);
CREATE INDEX idx_pending_contact_changes_expires_at ON auth.pending_contact_changes(expires_at);

-- External Accounts
CREATE TABLE auth.user_external_accounts (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    name VARCHAR(255),
    avatar_url VARCHAR(512),
    access_token TEXT,
    refresh_token TEXT,
    token_expires_at TIMESTAMPTZ,
    raw_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_external_accounts_user_id ON auth.user_external_accounts(user_id);
CREATE INDEX idx_external_accounts_provider_email ON auth.user_external_accounts(provider, email);

-- OAuth States
CREATE TABLE auth.oauth_states (
    state VARCHAR(255) PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    redirect_url VARCHAR(512),
    user_id VARCHAR(64),
    action VARCHAR(20) NOT NULL DEFAULT 'login',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_states_expires_at ON auth.oauth_states(expires_at);

-- Outbox table for transactional messaging
CREATE TABLE auth.outbox (
    id VARCHAR(36) PRIMARY KEY,
    event_name VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON auth.outbox (created_at) WHERE published_at IS NULL;




-- Auth Config
CREATE TABLE IF NOT EXISTS auth.auth_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- triggers for automatic updated_at
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.users FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.roles FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.permissions FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.role_permissions FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.user_roles FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.magic_codes FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.sessions FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.token_blacklist FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.pending_contact_changes FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.user_external_accounts FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.oauth_states FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.outbox FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();
CREATE TRIGGER set_updated_at BEFORE UPDATE ON auth.auth_config FOR EACH ROW EXECUTE FUNCTION auth.set_updated_at();

-- Reset search_path to default
SET search_path TO public;
