package hrm

import (
	"encoding/json"
	"time"
)

// --- Salary Structure DTOs ---

// AddSalaryStructureRequest represents request to add salary structure
type AddSalaryStructureRequest struct {
	EmployeeID    int64           `json:"employee_id" validate:"required"`
	BaseSalary    float64         `json:"base_salary" validate:"required,gt=0"`
	Currency      string          `json:"currency" validate:"required,len=3"`
	PayFrequency  string          `json:"pay_frequency" validate:"required,oneof=monthly bi_weekly weekly"`
	Allowances    json.RawMessage `json:"allowances,omitempty"`
	EffectiveFrom time.Time       `json:"effective_from" validate:"required"`
	EffectiveTo   *time.Time      `json:"effective_to,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
}

// EditSalaryStructureRequest represents request to edit salary structure
type EditSalaryStructureRequest struct {
	BaseSalary    *float64        `json:"base_salary,omitempty"`
	Currency      *string         `json:"currency,omitempty"`
	PayFrequency  *string         `json:"pay_frequency,omitempty"`
	Allowances    json.RawMessage `json:"allowances,omitempty"`
	EffectiveFrom *time.Time      `json:"effective_from,omitempty"`
	EffectiveTo   *time.Time      `json:"effective_to,omitempty"`
	Notes         *string         `json:"notes,omitempty"`
}

// SalaryStructureFilter represents filter for salary structures
type SalaryStructureFilter struct {
	EmployeeID  *int64     `json:"employee_id,omitempty"`
	ActiveOnly  bool       `json:"active_only,omitempty"`
	EffectiveAt *time.Time `json:"effective_at,omitempty"`
}

// --- Salary DTOs ---

// AddSalaryRequest represents request to create salary record
type AddSalaryRequest struct {
	EmployeeID int64   `json:"employee_id" validate:"required"`
	Year       int     `json:"year" validate:"required"`
	Month      int     `json:"month" validate:"required,min=1,max=12"`
	Notes      *string `json:"notes,omitempty"`
}

// EditSalaryRequest represents request to edit salary record
type EditSalaryRequest struct {
	BaseAmount       *float64 `json:"base_amount,omitempty"`
	AllowancesAmount *float64 `json:"allowances_amount,omitempty"`
	BonusesAmount    *float64 `json:"bonuses_amount,omitempty"`
	DeductionsAmount *float64 `json:"deductions_amount,omitempty"`
	TaxAmount        *float64 `json:"tax_amount,omitempty"`
	WorkedDays       *int     `json:"worked_days,omitempty"`
	TotalWorkDays    *int     `json:"total_work_days,omitempty"`
	OvertimeHours    *float64 `json:"overtime_hours,omitempty"`
	Notes            *string  `json:"notes,omitempty"`
}

// CalculateSalaryRequest represents request to calculate salary
type CalculateSalaryRequest struct {
	EmployeeID *int64 `json:"employee_id,omitempty"` // If not set, calculate for all
	Year       int    `json:"year" validate:"required"`
	Month      int    `json:"month" validate:"required,min=1,max=12"`
}

// ApproveSalaryRequest represents request to approve salary
type ApproveSalaryRequest struct {
	Approved bool    `json:"approved"`
	Notes    *string `json:"notes,omitempty"`
}

// SalaryFilter represents filter for salaries
type SalaryFilter struct {
	EmployeeID     *int64  `json:"employee_id,omitempty"`
	Year           *int    `json:"year,omitempty"`
	Month          *int    `json:"month,omitempty"`
	Status         *string `json:"status,omitempty"`
	DepartmentID   *int64  `json:"department_id,omitempty"`
	OrganizationID *int64  `json:"organization_id,omitempty"`
	Limit          int     `json:"limit,omitempty"`
	Offset         int     `json:"offset,omitempty"`
}

// --- Bonus DTOs ---

// AddBonusRequest represents request to add bonus
type AddBonusRequest struct {
	EmployeeID  int64   `json:"employee_id" validate:"required"`
	SalaryID    *int64  `json:"salary_id,omitempty"`
	BonusType   string  `json:"bonus_type" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description *string `json:"description,omitempty"`
	Year        *int    `json:"year,omitempty"`
	Month       *int    `json:"month,omitempty"`
}

// EditBonusRequest represents request to edit bonus
type EditBonusRequest struct {
	BonusType   *string  `json:"bonus_type,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	Description *string  `json:"description,omitempty"`
}

// ApproveBonusRequest represents request to approve bonus
type ApproveBonusRequest struct {
	Approved bool `json:"approved"`
}

// BonusFilter represents filter for bonuses
type BonusFilter struct {
	EmployeeID *int64  `json:"employee_id,omitempty"`
	SalaryID   *int64  `json:"salary_id,omitempty"`
	BonusType  *string `json:"bonus_type,omitempty"`
	Year       *int    `json:"year,omitempty"`
	Month      *int    `json:"month,omitempty"`
	Approved   *bool   `json:"approved,omitempty"`
}

// --- Deduction DTOs ---

// AddDeductionRequest represents request to add deduction
type AddDeductionRequest struct {
	EmployeeID     int64      `json:"employee_id" validate:"required"`
	SalaryID       *int64     `json:"salary_id,omitempty"`
	DeductionType  string     `json:"deduction_type" validate:"required"`
	Amount         float64    `json:"amount" validate:"required,gt=0"`
	Description    *string    `json:"description,omitempty"`
	Year           *int       `json:"year,omitempty"`
	Month          *int       `json:"month,omitempty"`
	IsRecurring    bool       `json:"is_recurring"`
	RecurringUntil *time.Time `json:"recurring_until,omitempty"`
}

// EditDeductionRequest represents request to edit deduction
type EditDeductionRequest struct {
	DeductionType  *string    `json:"deduction_type,omitempty"`
	Amount         *float64   `json:"amount,omitempty"`
	Description    *string    `json:"description,omitempty"`
	IsRecurring    *bool      `json:"is_recurring,omitempty"`
	RecurringUntil *time.Time `json:"recurring_until,omitempty"`
}

// DeductionFilter represents filter for deductions
type DeductionFilter struct {
	EmployeeID    *int64  `json:"employee_id,omitempty"`
	SalaryID      *int64  `json:"salary_id,omitempty"`
	DeductionType *string `json:"deduction_type,omitempty"`
	Year          *int    `json:"year,omitempty"`
	Month         *int    `json:"month,omitempty"`
	IsRecurring   *bool   `json:"is_recurring,omitempty"`
}
