-- Create schema for auth module
CREATE SCHEMA IF NOT EXISTS auth;

-- Set search_path for subsequent statements in this migration
SET search_path TO auth, public;

CREATE TABLE auth.users (
    id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(50) UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE auth.roles (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE auth.permissions (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL
);

CREATE TABLE auth.role_permissions (
    role_id VARCHAR(64) NOT NULL,
    permission_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES auth.roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES auth.permissions(id) ON DELETE CASCADE
);

CREATE TABLE auth.user_roles (
    user_id VARCHAR(64) NOT NULL,
    role_id VARCHAR(64) NOT NULL,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES auth.roles(id) ON DELETE CASCADE
);

CREATE TABLE auth.magic_codes (
    code VARCHAR(10) NOT NULL,
    user_email VARCHAR(255),
    user_phone VARCHAR(50),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_magic_codes_email ON auth.magic_codes(user_email);
CREATE INDEX idx_magic_codes_phone ON auth.magic_codes(user_phone);

-- Reset search_path to default
SET search_path TO public;
