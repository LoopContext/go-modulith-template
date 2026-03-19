// Package seed provides programmatic seeding for the auth module.
package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cmelgarejo/go-modulith-template/internal/audit"
	"github.com/cmelgarejo/go-modulith-template/internal/config"
	"github.com/cmelgarejo/go-modulith-template/modules/auth/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.jetify.com/typeid"
)

// Seed runs programmatic seed data for the auth module.
func Seed(ctx context.Context, dbPool *pgxpool.Pool, cfg *config.AppConfig, auditLogger audit.Logger) error {
	repo := repository.NewSQLRepository(dbPool)

	// 1. Seed Roles and Permissions
	if err := seedRolesAndPermissions(ctx, dbPool); err != nil {
		return fmt.Errorf("failed to seed roles and permissions: %w", err)
	}

	// 2. Seed Users
	for _, u := range cfg.Seeds.Users {
		if err := processUserSeed(ctx, dbPool, repo, auditLogger, u); err != nil {
			return err
		}
	}

	return nil
}

func seedRolesAndPermissions(ctx context.Context, db *pgxpool.Pool) error {
	permIDs, err := seedPermissions(ctx, db)
	if err != nil {
		return err
	}

	roleIDs, err := seedRoles(ctx, db)
	if err != nil {
		return err
	}

	return assignRolePermissions(ctx, db, roleIDs, permIDs)
}

func seedPermissions(ctx context.Context, db *pgxpool.Pool) (map[string]string, error) {
	perms := []struct {
		Name     string
		Resource string
		Action   string
	}{
		{"users:read", "users", "read"},
		{"users:write", "users", "write"},
		{"auth:debug", "auth", "debug"},
	}

	permIDs := make(map[string]string)

	for _, p := range perms {
		var id string

		err := db.QueryRow(ctx, "SELECT id FROM auth.permissions WHERE name = $1", p.Name).Scan(&id)
		if err != nil {
			tid, _ := typeid.WithPrefix("perm")
			id = tid.String()

			_, err = db.Exec(ctx, "INSERT INTO auth.permissions (id, name, resource, action) VALUES ($1, $2, $3, $4)", id, p.Name, p.Resource, p.Action)
			if err != nil {
				return nil, fmt.Errorf("failed to insert permission %s: %w", p.Name, err)
			}
		}

		permIDs[p.Name] = id
	}

	return permIDs, nil
}

func seedRoles(ctx context.Context, db *pgxpool.Pool) (map[string]string, error) {
	roles := []string{"user", "platform", "admin"}
	roleIDs := make(map[string]string)

	for _, r := range roles {
		var id string

		err := db.QueryRow(ctx, "SELECT id FROM auth.roles WHERE name = $1", r).Scan(&id)
		if err != nil {
			tid, _ := typeid.WithPrefix("role")
			id = tid.String()

			_, err = db.Exec(ctx, "INSERT INTO auth.roles (id, name) VALUES ($1, $2)", id, r)
			if err != nil {
				return nil, fmt.Errorf("failed to insert role %s: %w", r, err)
			}
		}

		roleIDs[r] = id
	}

	return roleIDs, nil
}

func assignRolePermissions(ctx context.Context, db *pgxpool.Pool, roleIDs, permIDs map[string]string) error {
	assignments := []struct {
		Role string
		Perm string
	}{
		{"user", "users:read"},
		{"platform", "users:read"},
		{"platform", "users:write"},
		{"platform", "auth:debug"},
		{"admin", "users:read"},
		{"admin", "users:write"},
		{"admin", "auth:debug"},
	}

	for _, a := range assignments {
		_, err := db.Exec(ctx, "INSERT INTO auth.role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", roleIDs[a.Role], permIDs[a.Perm])
		if err != nil {
			return fmt.Errorf("failed to assign permission %s to role %s: %w", a.Perm, a.Role, err)
		}
	}

	return nil
}

func processUserSeed(ctx context.Context, dbPool *pgxpool.Pool, repo *repository.SQLRepository, al audit.Logger, u config.SeedUser) error {
	// Check if user exists
	existing, _ := repo.GetUserByEmail(ctx, u.Email)
	if existing != nil {
		return assignUserRole(ctx, dbPool, al, existing.ID, u.Email, u.Role)
	}

	tid, err := typeid.WithPrefix("user")
	if err != nil {
		return fmt.Errorf("failed to generate user id: %w", err)
	}

	userID := tid.String()
	slog.Info("Seeding user", "email", u.Email, "id", userID)

	if err := repo.CreateUser(ctx, userID, u.Email, u.Phone); err != nil {
		return fmt.Errorf("failed to create user %s: %w", u.Email, err)
	}

	// Set default timezone for seeded users
	if err := repo.UpdateUserProfile(ctx, userID, "", "", "America/Asuncion"); err != nil {
		slog.Warn("failed to set default timezone for seeded user", "email", u.Email, "error", err)
	}

	// Audit Log: User Created
	al.Log(ctx, audit.LogParams{
		Action:     "user.created",
		Resource:   "users",
		ResourceID: userID,
		ActorID:    "system",
		Metadata: map[string]any{
			"email":  u.Email,
			"phone":  u.Phone,
			"source": "seed",
		},
		Success: true,
	})

	return assignUserRole(ctx, dbPool, al, userID, u.Email, u.Role)
}

func assignUserRole(ctx context.Context, dbPool *pgxpool.Pool, al audit.Logger, userID, email, roleName string) error {
	var roleID string

	err := dbPool.QueryRow(ctx, "SELECT id FROM auth.roles WHERE name = $1", roleName).Scan(&roleID)
	if err != nil {
		return fmt.Errorf("failed to find role %s: %w", roleName, err)
	}

	// Clear existing roles to ensure configuration is the single source of truth
	if _, err := dbPool.Exec(ctx, "DELETE FROM auth.user_roles WHERE user_id = $1", userID); err != nil {
		return fmt.Errorf("failed to clear existing roles for user %s: %w", email, err)
	}

	_, err = dbPool.Exec(ctx, "INSERT INTO auth.user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to assign role to user %s: %w", email, err)
	}

	// Audit Log: Role Assigned
	al.Log(ctx, audit.LogParams{
		Action:     "user.role_assigned",
		Resource:   "users",
		ResourceID: userID,
		ActorID:    "system",
		Metadata: map[string]any{
			"role":    roleName,
			"role_id": roleID,
			"source":  "seed",
		},
		Success: true,
	})

	return nil
}
