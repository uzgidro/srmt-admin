package startup_admin

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"srmt-admin/internal/lib/dto"
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
	AddUser(ctx context.Context, login string, passwordHash []byte, contactID int64) (int64, error)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (*user.Model, string, error)
	GetRoleByName(ctx context.Context, name string) (role.Model, error)
	AddRole(ctx context.Context, name string, description string) (int64, error)
	GetUserRoles(ctx context.Context, userID int64) ([]role.Model, error)
	AssignRole(ctx context.Context, userID, roleID int64) error
}

func EnsureAdminExists(ctx context.Context, log *slog.Logger, creator AdminCreator) error {
	const op = "setup.ensureAdminExists"
	log = log.With(slog.String("op", op))
	log.Info("checking for default admin user and role")

	// --- 1. Проверяем Роль ---
	r, err := creator.GetRoleByName(ctx, defaultAdminRole)
	if err != nil {
		if errors.Is(err, storage.ErrRoleNotFound) { // (Или storage.ErrNotFound)
			log.Info("admin role not found, creating it")
			newRoleID, createErr := creator.AddRole(ctx, defaultAdminRole, "Admin role")
			if createErr != nil {
				return createErr
			}
			r.ID = newRoleID
			r.Name = defaultAdminRole
		} else {
			// Ошибка при *поиске* роли (кроме "не найдено")
			return fmt.Errorf("%s: error on getting role: %w", op, err)
		}
	}

	// --- 2. Проверяем Пользователя ---
	u, _, err := creator.GetUserByLogin(ctx, defaultAdminName)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) { // (Или storage.ErrNotFound)
			log.Info("admin user not found, creating it")

			// --- (ИСПРАВЛЕНИЕ) ---
			// Создаем контакт ТОЛЬКО ЕСЛИ юзер не найден
			log.Info("creating default admin contact")
			contactID, contactErr := creator.AddContact(ctx, dto.AddContactRequest{FIO: "Администратор"})
			if contactErr != nil {
				// (Если `AddContact` упал, например, по unique_violation,
				//  тоже стоит проверить, возможно, контакт уже есть, а юзера нет?
				//  Но для простоты стартапа - это ОК)
				return fmt.Errorf("%s: error on creating admin contact: %w", op, contactErr)
			}
			// ---

			hashedPassword, passErr := bcrypt.GenerateFromPassword([]byte(defaultAdminPass), bcrypt.DefaultCost)
			if passErr != nil {
				return passErr
			}

			log.Info("creating default admin user")
			newUserID, createErr := creator.AddUser(ctx, defaultAdminName, hashedPassword, contactID)
			if createErr != nil {
				return createErr
			}

			// Мы не можем использовать 'u' из GetUserByLogin (он пустой),
			// нам нужно создать временный объект 'u' для следующего шага
			u = &user.Model{ID: newUserID, Login: defaultAdminName} // (Если у тебя user.Model - struct, а не *user.Model)
			// Если GetUserByLogin возвращает *user.Model:
			// u = &user.Model{ID: newUserID, Login: defaultAdminName}

		} else {
			// Ошибка при *поиске* юзера (кроме "не найдено")
			return fmt.Errorf("%s: error on getting user: %w", op, err)
		}
	}

	// --- 3. Проверяем и назначаем роль ---
	// (Теперь `u.ID` гарантированно существует, либо старый, либо новый)
	userRoles, err := creator.GetUserRoles(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("%s: error on getting user roles: %w", op, err)
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
			return fmt.Errorf("%s: assigning admin role to admin user: %w", op, err)
		}
	}

	log.Info("admin user and role are configured")
	return nil
}
