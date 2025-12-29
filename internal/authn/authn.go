// Package authn provides authentication helpers for gRPC (JWT verification + context injection).
package authn

import (
	"context"
)

type ctxKey string

const (
	ctxUserID ctxKey = "authn.user_id"
	ctxRole   ctxKey = "authn.role"
)

// Claims are the authenticated identity attributes extracted from a token.
type Claims struct {
	UserID string
	Role   string
}

// ContextWithClaims injects authentication claims into the context.
func ContextWithClaims(ctx context.Context, c Claims) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, c.UserID)
	ctx = context.WithValue(ctx, ctxRole, c.Role)

	return ctx
}

// UserIDFromContext extracts the authenticated user id from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxUserID).(string)
	if !ok || v == "" {
		return "", false
	}

	return v, true
}

// RoleFromContext extracts the authenticated role from context.
func RoleFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxRole).(string)
	if !ok || v == "" {
		return "", false
	}

	return v, true
}
