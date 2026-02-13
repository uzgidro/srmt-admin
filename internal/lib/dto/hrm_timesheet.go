package dto

// TimesheetFilters — query parameters for GET /hrm/timesheet
type TimesheetFilters struct {
	Year         int `validate:"required"`
	Month        int `validate:"required,min=1,max=12"`
	DepartmentID *int64
	EmployeeID   *int64
}

// CorrectionFilters — query parameters for GET /hrm/timesheet/corrections
type CorrectionFilters struct {
	EmployeeID *int64
	Status     *string
}

// UpdateTimesheetEntryRequest — PATCH /hrm/timesheet/:id
type UpdateTimesheetEntryRequest struct {
	Status      *string  `json:"status,omitempty" validate:"omitempty,oneof=present absent vacation sick_leave business_trip remote day_off holiday maternity study_leave unauthorized"`
	CheckIn     *string  `json:"check_in,omitempty"`
	CheckOut    *string  `json:"check_out,omitempty"`
	HoursWorked *float64 `json:"hours_worked,omitempty"`
	Overtime    *float64 `json:"overtime,omitempty"`
	Note        *string  `json:"note,omitempty"`
}

// CreateHolidayRequest — POST /hrm/holidays
type CreateHolidayRequest struct {
	Name        string  `json:"name" validate:"required"`
	Date        string  `json:"date" validate:"required"`
	Type        string  `json:"type" validate:"required,oneof=national religious company"`
	Description *string `json:"description,omitempty"`
}

// CreateTimesheetCorrectionRequest — POST /hrm/timesheet/corrections
type CreateTimesheetCorrectionRequest struct {
	EmployeeID  int64   `json:"employee_id" validate:"required"`
	Date        string  `json:"date" validate:"required"`
	NewStatus   string  `json:"new_status" validate:"required,oneof=present absent vacation sick_leave business_trip remote day_off holiday maternity study_leave unauthorized"`
	NewCheckIn  *string `json:"new_check_in,omitempty"`
	NewCheckOut *string `json:"new_check_out,omitempty"`
	Reason      string  `json:"reason" validate:"required"`
}

// RejectCorrectionRequest — POST /hrm/timesheet/corrections/:id/reject
type RejectCorrectionRequest struct {
	Reason string `json:"reason" validate:"required"`
}
