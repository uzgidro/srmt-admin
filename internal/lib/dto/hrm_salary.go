package dto

// CreateSalaryRequest — POST /hrm/salaries
type CreateSalaryRequest struct {
	EmployeeID  int64 `json:"employee_id" validate:"required"`
	PeriodMonth int   `json:"period_month" validate:"required,min=1,max=12"`
	PeriodYear  int   `json:"period_year" validate:"required,min=2020"`
}

// UpdateSalaryRequest — PATCH /hrm/salaries/:id
type UpdateSalaryRequest struct {
	WorkDays      *int     `json:"work_days,omitempty"`
	ActualDays    *int     `json:"actual_days,omitempty"`
	OvertimeHours *float64 `json:"overtime_hours,omitempty"`
}

// CalculateSalaryRequest — POST /hrm/salaries/:id/calculate
type CalculateSalaryRequest struct {
	WorkDays      int              `json:"work_days" validate:"required,min=1"`
	ActualDays    int              `json:"actual_days" validate:"required,min=0"`
	OvertimeHours float64          `json:"overtime_hours" validate:"min=0"`
	Bonuses       []BonusInput     `json:"bonuses,omitempty"`
	Deductions    []DeductionInput `json:"deductions,omitempty"`
}

// BulkCalculateRequest — POST /hrm/salaries/bulk-calculate
type BulkCalculateRequest struct {
	PeriodMonth  int    `json:"period_month" validate:"required,min=1,max=12"`
	PeriodYear   int    `json:"period_year" validate:"required,min=2020"`
	DepartmentID *int64 `json:"department_id,omitempty"`
}

// BonusInput — bonus item inside calculate request
type BonusInput struct {
	Type        string  `json:"type" validate:"required,oneof=performance quarterly annual holiday project overtime other"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description *string `json:"description,omitempty"`
}

// DeductionInput — deduction item inside calculate request
type DeductionInput struct {
	Type        string  `json:"type" validate:"required,oneof=tax pension insurance loan alimony fine advance other"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description *string `json:"description,omitempty"`
}

// SalaryFilters — query parameters for GET /hrm/salaries
type SalaryFilters struct {
	PeriodMonth  *int
	PeriodYear   *int
	DepartmentID *int64
	Status       *string
}
