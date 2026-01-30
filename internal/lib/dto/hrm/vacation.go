package hrm

import "time"

// --- Vacation Type DTOs ---

// AddVacationTypeRequest represents request to create vacation type
type AddVacationTypeRequest struct {
	Name               string  `json:"name" validate:"required"`
	Code               string  `json:"code" validate:"required"`
	Description        *string `json:"description,omitempty"`
	DefaultDaysPerYear int     `json:"default_days_per_year"`
	IsPaid             bool    `json:"is_paid"`
	RequiresApproval   bool    `json:"requires_approval"`
	CanCarryOver       bool    `json:"can_carry_over"`
	MaxCarryOverDays   int     `json:"max_carry_over_days"`
	SortOrder          int     `json:"sort_order"`
}

// EditVacationTypeRequest represents request to update vacation type
type EditVacationTypeRequest struct {
	Name               *string `json:"name,omitempty"`
	Code               *string `json:"code,omitempty"`
	Description        *string `json:"description,omitempty"`
	DefaultDaysPerYear *int    `json:"default_days_per_year,omitempty"`
	IsPaid             *bool   `json:"is_paid,omitempty"`
	RequiresApproval   *bool   `json:"requires_approval,omitempty"`
	CanCarryOver       *bool   `json:"can_carry_over,omitempty"`
	MaxCarryOverDays   *int    `json:"max_carry_over_days,omitempty"`
	IsActive           *bool   `json:"is_active,omitempty"`
	SortOrder          *int    `json:"sort_order,omitempty"`
}

// --- Vacation Balance DTOs ---

// AddVacationBalanceRequest represents request to set vacation balance
type AddVacationBalanceRequest struct {
	EmployeeID      int64   `json:"employee_id" validate:"required"`
	VacationTypeID  int     `json:"vacation_type_id" validate:"required"`
	Year            int     `json:"year" validate:"required"`
	EntitledDays    float64 `json:"entitled_days"`
	CarriedOverDays float64 `json:"carried_over_days"`
	AdjustmentDays  float64 `json:"adjustment_days"`
	Notes           *string `json:"notes,omitempty"`
}

// EditVacationBalanceRequest represents request to update vacation balance
type EditVacationBalanceRequest struct {
	EntitledDays    *float64 `json:"entitled_days,omitempty"`
	UsedDays        *float64 `json:"used_days,omitempty"`
	CarriedOverDays *float64 `json:"carried_over_days,omitempty"`
	AdjustmentDays  *float64 `json:"adjustment_days,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
}

// VacationBalanceFilter represents filter for vacation balances
type VacationBalanceFilter struct {
	EmployeeID     *int64 `json:"employee_id,omitempty"`
	VacationTypeID *int   `json:"vacation_type_id,omitempty"`
	Year           *int   `json:"year,omitempty"`
}

// --- Vacation Request DTOs ---

// AddVacationRequest represents request to create vacation/leave request
type AddVacationRequest struct {
	EmployeeID           int64     `json:"employee_id" validate:"required"`
	VacationTypeID       int       `json:"vacation_type_id" validate:"required"`
	StartDate            time.Time `json:"start_date" validate:"required"`
	EndDate              time.Time `json:"end_date" validate:"required"`
	DaysCount            float64   `json:"days_count" validate:"required,gt=0"`
	Reason               *string   `json:"reason,omitempty"`
	SubstituteEmployeeID *int64    `json:"substitute_employee_id,omitempty"`
	SupportingDocumentID *int64    `json:"supporting_document_id,omitempty"`
}

// EditVacationRequest represents request to edit vacation request
type EditVacationRequest struct {
	VacationTypeID       *int       `json:"vacation_type_id,omitempty"`
	StartDate            *time.Time `json:"start_date,omitempty"`
	EndDate              *time.Time `json:"end_date,omitempty"`
	DaysCount            *float64   `json:"days_count,omitempty"`
	Reason               *string    `json:"reason,omitempty"`
	SubstituteEmployeeID *int64     `json:"substitute_employee_id,omitempty"`
	SupportingDocumentID *int64     `json:"supporting_document_id,omitempty"`
}

// ApproveVacationRequest represents request to approve/reject vacation
type ApproveVacationRequest struct {
	Approved        bool    `json:"approved"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// VacationFilter represents filter for vacation requests
type VacationFilter struct {
	EmployeeID     *int64     `json:"employee_id,omitempty"`
	VacationTypeID *int       `json:"vacation_type_id,omitempty"`
	Status         *string    `json:"status,omitempty"`
	FromDate       *time.Time `json:"from_date,omitempty"`
	ToDate         *time.Time `json:"to_date,omitempty"`
	DepartmentID   *int64     `json:"department_id,omitempty"`
	Limit          int        `json:"limit,omitempty"`
	Offset         int        `json:"offset,omitempty"`
}

// VacationCalendarFilter represents filter for vacation calendar
type VacationCalendarFilter struct {
	Year           int    `json:"year" validate:"required"`
	Month          int    `json:"month" validate:"required,min=1,max=12"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	OrganizationID *int64 `json:"organization_id,omitempty"`
}
