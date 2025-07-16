package storage

import (
	"database/sql"
	"errors"
)

type Driver struct {
	DB         *sql.DB
	Translator ErrorTranslator
}

type ErrorTranslator interface {
	Translate(err error, op string) error
}

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrDuplicate           = errors.New("duplicate entry")
	ErrRoleNotFound        = errors.New("role not found")
	ErrForeignKeyViolation = errors.New("provided id for user or role does not exist")
)
