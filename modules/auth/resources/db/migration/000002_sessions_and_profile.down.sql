-- Remove pending contact changes
DROP TABLE IF EXISTS pending_contact_changes;

-- Remove token blacklist
DROP TABLE IF EXISTS token_blacklist;

-- Remove sessions
DROP TABLE IF EXISTS sessions;

-- Remove profile fields from users
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS display_name;

