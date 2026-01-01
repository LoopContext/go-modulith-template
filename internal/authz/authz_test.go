package authz

import (
	"context"
	"testing"

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/errors"
)

func TestRequirePermission(t *testing.T) {
	t.Run("admin has all permissions", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "admin",
		})

		err := RequirePermission(ctx, PermissionUsersDelete)
		if err != nil {
			t.Errorf("expected admin to have permission, got error: %v", err)
		}
	})

	t.Run("user has read permission", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "user",
		})

		err := RequirePermission(ctx, PermissionUsersRead)
		if err != nil {
			t.Errorf("expected user to have read permission, got error: %v", err)
		}
	})

	t.Run("user lacks write permission", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "user",
		})

		err := RequirePermission(ctx, PermissionUsersWrite)
		if err == nil {
			t.Error("expected error for missing permission")
		}

		if !errors.IsValidation(err) && err.Error() != "permission denied: users:write" {
			// Check if it's a forbidden error
			if err.Error() != "permission denied: users:write" {
				t.Errorf("expected forbidden error, got: %v", err)
			}
		}
	})

	t.Run("unauthenticated user", func(t *testing.T) {
		ctx := context.Background()

		err := RequirePermission(ctx, PermissionUsersRead)
		if err == nil {
			t.Error("expected error for unauthenticated user")
		}
	})
}

func TestRequireRole(t *testing.T) {
	t.Run("user has allowed role", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "admin",
		})

		err := RequireRole(ctx, "admin", "moderator")
		if err != nil {
			t.Errorf("expected user to have allowed role, got error: %v", err)
		}
	})

	t.Run("user lacks allowed role", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "user",
		})

		err := RequireRole(ctx, "admin", "moderator")
		if err == nil {
			t.Error("expected error for missing role")
		}
	})
}

func TestRequireOwnership(t *testing.T) {
	t.Run("user owns resource", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "user",
		})

		err := RequireOwnership(ctx, "user_123")
		if err != nil {
			t.Errorf("expected user to own resource, got error: %v", err)
		}
	})

	t.Run("user does not own resource", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "user",
		})

		err := RequireOwnership(ctx, "user_456")
		if err == nil {
			t.Error("expected error for non-owner")
		}
	})

	t.Run("admin can access any resource", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "admin_123",
			Role:   "admin",
		})

		err := RequireOwnership(ctx, "user_456")
		if err != nil {
			t.Errorf("expected admin to access resource, got error: %v", err)
		}
	})
}

func TestRequireOwnershipOrRole(t *testing.T) {
	t.Run("user owns resource", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "user",
		})

		err := RequireOwnershipOrRole(ctx, "user_123", "admin")
		if err != nil {
			t.Errorf("expected user to own resource, got error: %v", err)
		}
	})

	t.Run("user has allowed role", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "admin",
		})

		err := RequireOwnershipOrRole(ctx, "user_456", "admin")
		if err != nil {
			t.Errorf("expected user to have allowed role, got error: %v", err)
		}
	})

	t.Run("user neither owns nor has role", func(t *testing.T) {
		ctx := authn.ContextWithClaims(context.Background(), authn.Claims{
			UserID: "user_123",
			Role:   "user",
		})

		err := RequireOwnershipOrRole(ctx, "user_456", "admin")
		if err == nil {
			t.Error("expected error for non-owner without role")
		}
	})
}

func TestRegisterRole(t *testing.T) {
	t.Run("register custom role", func(t *testing.T) {
		RegisterRole("moderator", []Permission{"posts:delete", "comments:delete"})

		perms := GetRolePermissions("moderator")
		if len(perms) != 2 {
			t.Errorf("expected 2 permissions, got %d", len(perms))
		}
	})
}

func TestHasPermission(t *testing.T) {
	t.Run("admin has wildcard permission", func(t *testing.T) {
		if !HasPermission("admin", "anything:action") {
			t.Error("expected admin to have wildcard permission")
		}
	})

	t.Run("user has specific permission", func(t *testing.T) {
		if !HasPermission("user", PermissionUsersRead) {
			t.Error("expected user to have read permission")
		}
	})

	t.Run("user lacks permission", func(t *testing.T) {
		if HasPermission("user", PermissionUsersDelete) {
			t.Error("expected user to lack delete permission")
		}
	})

	t.Run("unknown role", func(t *testing.T) {
		if HasPermission("unknown", PermissionUsersRead) {
			t.Error("expected unknown role to have no permissions")
		}
	})
}

