-- name: CreateUser :exec
INSERT INTO users (id, email, phone) VALUES ($1, $2, $3);

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1 LIMIT 1;

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
