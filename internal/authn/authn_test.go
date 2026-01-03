package authn

import (
	"context"
	"testing"
)

func TestContextWithClaims(t *testing.T) {
	ctx := context.Background()
	claims := Claims{
		UserID: "user-123",
		Role:   "admin",
	}

	ctx = ContextWithClaims(ctx, claims)

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		t.Fatal("expected user ID to be in context")
	}

	if userID != claims.UserID {
		t.Errorf("expected user ID %s, got %s", claims.UserID, userID)
	}

	role, ok := RoleFromContext(ctx)
	if !ok {
		t.Fatal("expected role to be in context")
	}

	if role != claims.Role {
		t.Errorf("expected role %s, got %s", claims.Role, role)
	}
}

func TestUserIDFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	userID, ok := UserIDFromContext(ctx)
	if ok {
		t.Error("expected user ID to not be in context")
	}

	if userID != "" {
		t.Errorf("expected empty user ID, got %s", userID)
	}
}

func TestUserIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxUserID, "")

	userID, ok := UserIDFromContext(ctx)
	if ok {
		t.Error("expected user ID to not be valid when empty")
	}

	if userID != "" {
		t.Errorf("expected empty user ID, got %s", userID)
	}
}

func TestRoleFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	role, ok := RoleFromContext(ctx)
	if ok {
		t.Error("expected role to not be in context")
	}

	if role != "" {
		t.Errorf("expected empty role, got %s", role)
	}
}

func TestRoleFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxRole, "")

	role, ok := RoleFromContext(ctx)
	if ok {
		t.Error("expected role to not be valid when empty")
	}

	if role != "" {
		t.Errorf("expected empty role, got %s", role)
	}
}

func TestRoleFromContext_WrongType(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxRole, 123)

	role, ok := RoleFromContext(ctx)
	if ok {
		t.Error("expected role to not be valid when wrong type")
	}

	if role != "" {
		t.Errorf("expected empty role, got %s", role)
	}
}

func TestUserIDFromContext_WrongType(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxUserID, 123)

	userID, ok := UserIDFromContext(ctx)
	if ok {
		t.Error("expected user ID to not be valid when wrong type")
	}

	if userID != "" {
		t.Errorf("expected empty user ID, got %s", userID)
	}
}
