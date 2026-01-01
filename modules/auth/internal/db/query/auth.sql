-- ========================
-- User Management
-- ========================

-- name: CreateUser :exec
INSERT INTO users (id, email, phone) VALUES ($1, $2, $3);

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1 LIMIT 1;

-- name: UpdateUserProfile :exec
UPDATE users SET display_name = $2, avatar_url = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $1;

-- ========================
-- Magic Codes (Passwordless)
-- ========================

-- name: CreateMagicCode :exec
INSERT INTO magic_codes (code, user_email, user_phone, expires_at) VALUES ($1, $2, $3, $4);

-- name: GetValidMagicCodeByEmail :one
SELECT * FROM magic_codes
WHERE user_email = $1 AND code = $2 AND expires_at > $3
ORDER BY created_at DESC LIMIT 1;

-- name: GetValidMagicCodeByPhone :one
SELECT * FROM magic_codes
WHERE user_phone = $1 AND code = $2 AND expires_at > $3
ORDER BY created_at DESC LIMIT 1;

-- name: DeleteMagicCodesByEmail :exec
DELETE FROM magic_codes WHERE user_email = $1;

-- name: DeleteMagicCodesByPhone :exec
DELETE FROM magic_codes WHERE user_phone = $1;

-- name: CleanupExpiredMagicCodes :exec
DELETE FROM magic_codes WHERE expires_at < CURRENT_TIMESTAMP;

-- ========================
-- Sessions
-- ========================

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1 AND revoked_at IS NULL LIMIT 1;

-- name: GetSessionByRefreshTokenHash :one
SELECT * FROM sessions WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP LIMIT 1;

-- name: GetSessionsByUserID :many
SELECT * FROM sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP ORDER BY last_active_at DESC;

-- name: UpdateSessionActivity :exec
UPDATE sessions SET last_active_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: RevokeAllUserSessions :execrows
UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND revoked_at IS NULL AND ($2 = '' OR id != $2);

-- name: CleanupExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '7 days';

-- ========================
-- Token Blacklist
-- ========================

-- name: BlacklistToken :exec
INSERT INTO token_blacklist (token_hash, user_id, expires_at, reason)
VALUES ($1, $2, $3, $4)
ON CONFLICT (token_hash) DO NOTHING;

-- name: IsTokenBlacklisted :one
SELECT EXISTS(SELECT 1 FROM token_blacklist WHERE token_hash = $1 AND expires_at > CURRENT_TIMESTAMP);

-- name: CleanupExpiredBlacklistEntries :exec
DELETE FROM token_blacklist WHERE expires_at < CURRENT_TIMESTAMP;

-- ========================
-- Pending Contact Changes
-- ========================

-- name: CreatePendingContactChange :exec
INSERT INTO pending_contact_changes (id, user_id, change_type, new_value, verification_code, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetPendingContactChange :one
SELECT * FROM pending_contact_changes
WHERE user_id = $1 AND change_type = $2 AND verification_code = $3 AND expires_at > CURRENT_TIMESTAMP
LIMIT 1;

-- name: DeletePendingContactChange :exec
DELETE FROM pending_contact_changes WHERE id = $1;

-- name: DeleteExpiredPendingContactChanges :exec
DELETE FROM pending_contact_changes WHERE expires_at < CURRENT_TIMESTAMP;

-- ========================
-- External OAuth Accounts
-- ========================

-- name: CreateExternalAccount :exec
INSERT INTO user_external_accounts (id, user_id, provider, provider_user_id, email, name, avatar_url, access_token, refresh_token, token_expires_at, raw_data)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetExternalAccountByProviderAndUserID :one
SELECT * FROM user_external_accounts WHERE provider = $1 AND provider_user_id = $2 LIMIT 1;

-- name: GetExternalAccountsByUserID :many
SELECT * FROM user_external_accounts WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetExternalAccountByProviderAndEmail :one
SELECT * FROM user_external_accounts WHERE provider = $1 AND email = $2 LIMIT 1;

-- name: UpdateExternalAccountTokens :exec
UPDATE user_external_accounts
SET access_token = $3, refresh_token = $4, token_expires_at = $5, updated_at = CURRENT_TIMESTAMP
WHERE provider = $1 AND provider_user_id = $2;

-- name: UpdateExternalAccountProfile :exec
UPDATE user_external_accounts
SET name = $3, avatar_url = $4, email = $5, raw_data = $6, updated_at = CURRENT_TIMESTAMP
WHERE provider = $1 AND provider_user_id = $2;

-- name: DeleteExternalAccount :exec
DELETE FROM user_external_accounts WHERE id = $1 AND user_id = $2;

-- name: DeleteExternalAccountByProvider :exec
DELETE FROM user_external_accounts WHERE user_id = $1 AND provider = $2;

-- name: CountExternalAccountsByUserID :one
SELECT COUNT(*) FROM user_external_accounts WHERE user_id = $1;

-- ========================
-- OAuth State Tokens
-- ========================

-- name: CreateOAuthState :exec
INSERT INTO oauth_states (state, provider, redirect_url, user_id, action, expires_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetOAuthState :one
SELECT * FROM oauth_states WHERE state = $1 AND expires_at > CURRENT_TIMESTAMP LIMIT 1;

-- name: DeleteOAuthState :exec
DELETE FROM oauth_states WHERE state = $1;

-- name: CleanupExpiredOAuthStates :exec
DELETE FROM oauth_states WHERE expires_at < CURRENT_TIMESTAMP;
