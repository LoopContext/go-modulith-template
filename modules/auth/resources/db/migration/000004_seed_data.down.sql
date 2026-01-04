-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

-- Remove test data
DELETE FROM auth.user_roles;
DELETE FROM auth.role_permissions;
DELETE FROM auth.permissions;
DELETE FROM auth.roles;
DELETE FROM auth.users;

-- Reset search_path to default
SET search_path TO public;

