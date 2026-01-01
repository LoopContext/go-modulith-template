// Package authz provides role-based access control (RBAC) helpers.
package authz

import (
	"context"
	"fmt"

	"github.com/cmelgarejo/go-modulith-template/internal/authn"
	"github.com/cmelgarejo/go-modulith-template/internal/errors"
)

// Permission represents a permission string (e.g., "users:read", "orders:write").
type Permission string

// Common permissions
const (
	PermissionUsersRead   Permission = "users:read"
	PermissionUsersWrite  Permission = "users:write"
	PermissionUsersDelete Permission = "users:delete"
	PermissionAdminAll    Permission = "admin:*"
)

// Role represents a user role with associated permissions.
type Role struct {
	Name        string
	Permissions []Permission
}

// Common roles
var (
	RoleAdmin = Role{
		Name: "admin",
		Permissions: []Permission{
			PermissionAdminAll,
		},
	}

	RoleUser = Role{
		Name: "user",
		Permissions: []Permission{
			PermissionUsersRead,
		},
	}
)

// rolePermissions maps role names to their permissions.
var rolePermissions = map[string][]Permission{
	"admin": {PermissionAdminAll},
	"user":  {PermissionUsersRead},
}

// RequirePermission checks if the authenticated user has the required permission.
// Returns an error if the user is not authenticated or lacks the permission.
//
//nolint:wrapcheck // Domain errors are intentionally returned unwrapped for direct use
func RequirePermission(ctx context.Context, permission Permission) error {
	role, ok := authn.RoleFromContext(ctx)
	if !ok {
		return errors.Unauthorized("authentication required")
	}

	if HasPermission(role, permission) {
		return nil
	}

	return errors.Forbidden(fmt.Sprintf("permission denied: %s", permission))
}

// HasPermission checks if a role has a specific permission.
func HasPermission(roleName string, permission Permission) bool {
	permissions, ok := rolePermissions[roleName]
	if !ok {
		return false
	}

	// Check for wildcard admin permission
	for _, p := range permissions {
		if p == PermissionAdminAll {
			return true
		}

		if p == permission {
			return true
		}
	}

	return false
}

// RequireRole checks if the authenticated user has one of the required roles.
//
//nolint:wrapcheck // Domain errors are intentionally returned unwrapped for direct use
func RequireRole(ctx context.Context, allowedRoles ...string) error {
	role, ok := authn.RoleFromContext(ctx)
	if !ok {
		return errors.Unauthorized("authentication required")
	}

	for _, allowedRole := range allowedRoles {
		if role == allowedRole {
			return nil
		}
	}

	return errors.Forbidden("insufficient permissions")
}

// RequireOwnership checks if the authenticated user owns the resource.
// The resourceOwnerID should be the user ID who owns the resource.
//
//nolint:wrapcheck // Domain errors are intentionally returned unwrapped for direct use
func RequireOwnership(ctx context.Context, resourceOwnerID string) error {
	userID, ok := authn.UserIDFromContext(ctx)
	if !ok {
		return errors.Unauthorized("authentication required")
	}

	// Admin can access anything
	role, _ := authn.RoleFromContext(ctx)
	if role == "admin" {
		return nil
	}

	if userID != resourceOwnerID {
		return errors.Forbidden("access denied: not the resource owner")
	}

	return nil
}

// RequireOwnershipOrRole checks if the user owns the resource OR has one of the allowed roles.
//
//nolint:wrapcheck // Domain errors are intentionally returned unwrapped for direct use
func RequireOwnershipOrRole(ctx context.Context, resourceOwnerID string, allowedRoles ...string) error {
	userID, ok := authn.UserIDFromContext(ctx)
	if !ok {
		return errors.Unauthorized("authentication required")
	}

	// Check ownership first
	if userID == resourceOwnerID {
		return nil
	}

	// Check if user has allowed role
	role, ok := authn.RoleFromContext(ctx)
	if !ok {
		return errors.Forbidden("access denied")
	}

	for _, allowedRole := range allowedRoles {
		if role == allowedRole {
			return nil
		}
	}

	return errors.Forbidden("access denied: not the resource owner and insufficient role")
}

// RegisterRole registers a new role with its permissions.
// This allows modules to define custom roles.
func RegisterRole(roleName string, permissions []Permission) {
	rolePermissions[roleName] = permissions
}

// GetRolePermissions returns the permissions for a given role.
func GetRolePermissions(roleName string) []Permission {
	return rolePermissions[roleName]
}

