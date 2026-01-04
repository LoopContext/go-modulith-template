-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

-- Remove pending contact changes
DROP TABLE IF EXISTS auth.pending_contact_changes;

-- Remove token blacklist
DROP TABLE IF EXISTS auth.token_blacklist;

-- Remove sessions
DROP TABLE IF EXISTS auth.sessions;

-- Remove profile fields from users
ALTER TABLE auth.users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE auth.users DROP COLUMN IF EXISTS display_name;

-- Reset search_path to default
SET search_path TO public;

