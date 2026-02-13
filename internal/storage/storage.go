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

	// HRM errors
	ErrPersonnelRecordNotFound = errors.New("personnel record not found")
	ErrVacationNotFound        = errors.New("vacation not found")
	ErrBalanceNotFound         = errors.New("vacation balance not found")
	ErrInsufficientBalance     = errors.New("insufficient vacation balance")
	ErrVacationOverlap         = errors.New("vacation dates overlap with existing vacation")
	ErrBlockedPeriod           = errors.New("vacation dates fall within a blocked period")
	ErrInvalidDateRange        = errors.New("invalid date range")
	ErrStartDateInPast         = errors.New("start date cannot be in the past")
	ErrNotificationNotFound    = errors.New("notification not found")

	// Timesheet errors
	ErrTimesheetEntryNotFound = errors.New("timesheet entry not found")
	ErrHolidayNotFound        = errors.New("holiday not found")
	ErrHolidayAlreadyExists   = errors.New("holiday already exists for this date")
	ErrCorrectionNotFound     = errors.New("timesheet correction not found")

	// Salary errors
	ErrSalaryNotFound          = errors.New("salary record not found")
	ErrSalaryStructureNotFound = errors.New("salary structure not found")
	ErrSalaryAlreadyExists     = errors.New("salary record already exists for this period")
)
