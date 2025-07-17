package startup_admin

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"srmt-admin/internal/lib/model/role"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
)

const (
	defaultAdminName = "admin"
	defaultAdminPass = "admin123"
	defaultAdminRole = "admin"
)

type AdminCreator interface {
	AddUser(ctx context.Context, name, passHash string) (int64, error)
	GetUserByName(ctx context.Context, name string) (user.Model, error)
	GetRoleByName(ctx context.Context, name string) (role.Model, error)
	AddRole(ctx context.Context, name string, description string) (int64, error)
	GetUserRoles(ctx context.Context, userID int64) ([]role.Model, error)
	AssignRole(ctx context.Context, userID, roleID int64) error
}

func EnsureAdminExists(ctx context.Context, log *slog.Logger, creator AdminCreator) error {
	const op = "setup.ensureAdminExists"
	log = log.With(slog.String("op", op))
	log.Info("checking for default admin user and role")

	r, err := creator.GetRoleByName(ctx, defaultAdminRole)
	if err != nil {
		if errors.Is(err, storage.ErrRoleNotFound) {
			log.Info("admin role not found, creating it")
			newRoleID, createErr := creator.AddRole(ctx, defaultAdminRole, "Admin role")
			if createErr != nil {
				return createErr
			}
			r.ID = newRoleID
			r.Name = defaultAdminRole
		} else {
			return fmt.Errorf("%s: error on creating role %w", op, err)
		}
	}

	u, err := creator.GetUserByName(ctx, defaultAdminName)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Info("admin user not found, creating it")

			hashedPassword, passErr := bcrypt.GenerateFromPassword([]byte(defaultAdminPass), bcrypt.DefaultCost)
			if passErr != nil {
				return passErr
			}

			newUserID, createErr := creator.AddUser(ctx, defaultAdminName, string(hashedPassword))
			if createErr != nil {
				return createErr
			}
			u.ID = newUserID
			u.Name = defaultAdminName
		} else {
			return fmt.Errorf("%s: error on creating user %w", op, err)
		}
	}

	userRoles, err := creator.GetUserRoles(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("%s: error on connecting user with role %w", op, err)
	}

	var hasAdminRole bool
	for _, userRole := range userRoles {
		if userRole.ID == r.ID {
			hasAdminRole = true
			break
		}
	}

	if !hasAdminRole {
		log.Info("assigning admin role to admin user")
		if err := creator.AssignRole(ctx, u.ID, r.ID); err != nil {
			return fmt.Errorf("%s: assigning admin role to admin user %w", op, err)
		}
	}

	log.Info("admin user and role are configured")
	return nil
}
