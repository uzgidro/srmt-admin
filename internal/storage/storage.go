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

	ErrIndicatorNotFound    = errors.New("indicator not found")
	ErrLevelOutOfCurveRange = errors.New("level is outside the defined curve range")

	ErrDataNotFound      = errors.New("data not found")
	ErrSnowDataNotFound  = errors.New("snow data not found")
	ErrStockDataNotFound = errors.New("stock data not found")

	ErrUniqueViolation = errors.New("unique violation")

	ErrNotFound = errors.New("not found")
)
