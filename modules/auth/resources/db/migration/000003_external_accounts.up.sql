-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

-- External OAuth Accounts
-- Links users to external OAuth providers (Google, Facebook, GitHub, etc.)

CREATE TABLE auth.user_external_accounts (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    provider VARCHAR(50) NOT NULL,           -- google, facebook, github, apple, microsoft, twitter
    provider_user_id VARCHAR(255) NOT NULL,  -- ID of the user in the external provider
    email VARCHAR(255),                       -- Email from the provider (may differ from user's email)
    name VARCHAR(255),                        -- Display name from the provider
    avatar_url VARCHAR(512),                  -- Avatar/profile picture URL
    access_token TEXT,                        -- Encrypted access token
    refresh_token TEXT,                       -- Encrypted refresh token
    token_expires_at TIMESTAMP,               -- When the access token expires
    raw_data JSONB,                          -- Additional data from the provider
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    UNIQUE (provider, provider_user_id)
);

-- Index for looking up accounts by user
CREATE INDEX idx_external_accounts_user_id ON auth.user_external_accounts(user_id);

-- Index for looking up accounts by provider and email (for auto-linking)
CREATE INDEX idx_external_accounts_provider_email ON auth.user_external_accounts(provider, email);

-- OAuth State tokens for CSRF protection
CREATE TABLE auth.oauth_states (
    state VARCHAR(255) PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    redirect_url VARCHAR(512),
    user_id VARCHAR(64),                      -- NULL for login, set for account linking
    action VARCHAR(20) NOT NULL DEFAULT 'login', -- 'login' or 'link'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

-- Index for cleanup of expired states
CREATE INDEX idx_oauth_states_expires_at ON auth.oauth_states(expires_at);

-- Reset search_path to default
SET search_path TO public;

