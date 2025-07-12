package storage

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserExists          = errors.New("user with this name already exists")
	ErrRoleExists          = errors.New("role already exists")
	ErrRoleNotFound        = errors.New("role not found")
	ErrForeignKeyViolation = errors.New("provided id for user or role does not exist")
)
