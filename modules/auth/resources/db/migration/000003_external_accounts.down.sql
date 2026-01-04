-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

-- Drop OAuth state tokens table
DROP TABLE IF EXISTS auth.oauth_states;

-- Drop external accounts table
DROP TABLE IF EXISTS auth.user_external_accounts;

-- Reset search_path to default
SET search_path TO public;

