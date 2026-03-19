-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

DROP FUNCTION IF EXISTS auth.set_updated_at();

DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS user_external_accounts;
DROP TABLE IF EXISTS pending_contact_changes;
DROP TABLE IF EXISTS token_blacklist;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS magic_codes;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;

-- Reset search_path to default
SET search_path TO public;

-- Drop schema
DROP SCHEMA IF EXISTS auth CASCADE;
