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

	ErrNotFound      = errors.New("not found")
	ErrInvalidStatus = errors.New("invalid status for operation")

	// HRM-specific errors
	ErrInsufficientBalance      = errors.New("insufficient vacation balance")
	ErrVacationOverlap          = errors.New("vacation dates overlap with existing request")
	ErrCircularManagerHierarchy = errors.New("circular manager hierarchy detected")
	ErrSubstituteCannotBeSelf   = errors.New("substitute employee cannot be the same as requesting employee")
	ErrNegativeNetAmount        = errors.New("net salary amount cannot be negative")
	ErrInsufficientVacationDays = errors.New("requested days exceed available vacation balance")
	ErrAccessDenied             = errors.New("access denied to requested resource")
)
