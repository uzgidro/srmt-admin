package dto

type CreateVacationRequest struct {
	EmployeeID   int64   `json:"employee_id" validate:"required"`
	VacationType string  `json:"vacation_type" validate:"required,oneof=annual additional study unpaid maternity comp"`
	StartDate    string  `json:"start_date" validate:"required"`
	EndDate      string  `json:"end_date" validate:"required"`
	Reason       *string `json:"reason,omitempty"`
	SubstituteID *int64  `json:"substitute_id,omitempty"`
}

type EditVacationRequest struct {
	VacationType *string `json:"vacation_type,omitempty" validate:"omitempty,oneof=annual additional study unpaid maternity comp"`
	StartDate    *string `json:"start_date,omitempty"`
	EndDate      *string `json:"end_date,omitempty"`
	Reason       *string `json:"reason,omitempty"`
	SubstituteID *int64  `json:"substitute_id,omitempty"`
}

type RejectVacationRequest struct {
	Reason string `json:"reason" validate:"required"`
}

type VacationFilters struct {
	EmployeeID   *int64
	Status       *string
	VacationType *string
	StartDate    *string
	EndDate      *string
}

type VacationCalendarFilters struct {
	DepartmentID *int64
	StartDate    *string
	EndDate      *string
}
