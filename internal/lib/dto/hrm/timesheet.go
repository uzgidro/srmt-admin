package hrm

import "time"

// --- Holiday DTOs ---

// AddHolidayRequest represents request to add holiday
type AddHolidayRequest struct {
	Name         string    `json:"name" validate:"required"`
	Date         time.Time `json:"date" validate:"required"`
	Year         int       `json:"year" validate:"required"`
	IsWorkingDay bool      `json:"is_working_day"`
}

// EditHolidayRequest represents request to edit holiday
type EditHolidayRequest struct {
	Name         *string    `json:"name,omitempty"`
	Date         *time.Time `json:"date,omitempty"`
	IsWorkingDay *bool      `json:"is_working_day,omitempty"`
}

// HolidayFilter represents filter for holidays
type HolidayFilter struct {
	Year *int `json:"year,omitempty"`
}

// --- Timesheet DTOs ---

// AddTimesheetRequest represents request to create timesheet
type AddTimesheetRequest struct {
	EmployeeID int64   `json:"employee_id" validate:"required"`
	Year       int     `json:"year" validate:"required"`
	Month      int     `json:"month" validate:"required,min=1,max=12"`
	Notes      *string `json:"notes,omitempty"`
}

// EditTimesheetRequest represents request to edit timesheet
type EditTimesheetRequest struct {
	TotalWorkDays   *int     `json:"total_work_days,omitempty"`
	TotalWorkedDays *int     `json:"total_worked_days,omitempty"`
	TotalHours      *float64 `json:"total_hours,omitempty"`
	OvertimeHours   *float64 `json:"overtime_hours,omitempty"`
	SickDays        *int     `json:"sick_days,omitempty"`
	VacationDays    *int     `json:"vacation_days,omitempty"`
	AbsenceDays     *int     `json:"absence_days,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
}

// SubmitTimesheetRequest represents request to submit timesheet
type SubmitTimesheetRequest struct {
	Notes *string `json:"notes,omitempty"`
}

// ApproveTimesheetRequest represents request to approve/reject timesheet
type ApproveTimesheetRequest struct {
	Approved        bool    `json:"approved"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// TimesheetFilter represents filter for timesheets
type TimesheetFilter struct {
	EmployeeID     *int64  `json:"employee_id,omitempty"`
	Year           *int    `json:"year,omitempty"`
	Month          *int    `json:"month,omitempty"`
	Status         *string `json:"status,omitempty"`
	DepartmentID   *int64  `json:"department_id,omitempty"`
	OrganizationID *int64  `json:"organization_id,omitempty"`
	Limit          int     `json:"limit,omitempty"`
	Offset         int     `json:"offset,omitempty"`
}

// --- Timesheet Entry DTOs ---

// AddTimesheetEntryRequest represents request to add timesheet entry
type AddTimesheetEntryRequest struct {
	TimesheetID   int64     `json:"timesheet_id" validate:"required"`
	EmployeeID    int64     `json:"employee_id" validate:"required"`
	Date          time.Time `json:"date" validate:"required"`
	CheckIn       *string   `json:"check_in,omitempty"` // HH:MM format
	CheckOut      *string   `json:"check_out,omitempty"`
	BreakMinutes  int       `json:"break_minutes"`
	WorkedHours   float64   `json:"worked_hours"`
	OvertimeHours float64   `json:"overtime_hours"`
	DayType       string    `json:"day_type" validate:"required,oneof=work weekend holiday vacation sick absence remote"`
	IsRemote      bool      `json:"is_remote"`
	Notes         *string   `json:"notes,omitempty"`
}

// EditTimesheetEntryRequest represents request to edit timesheet entry
type EditTimesheetEntryRequest struct {
	CheckIn       *string  `json:"check_in,omitempty"`
	CheckOut      *string  `json:"check_out,omitempty"`
	BreakMinutes  *int     `json:"break_minutes,omitempty"`
	WorkedHours   *float64 `json:"worked_hours,omitempty"`
	OvertimeHours *float64 `json:"overtime_hours,omitempty"`
	DayType       *string  `json:"day_type,omitempty"`
	IsRemote      *bool    `json:"is_remote,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
}

// TimesheetEntryFilter represents filter for timesheet entries
type TimesheetEntryFilter struct {
	TimesheetID *int64     `json:"timesheet_id,omitempty"`
	EmployeeID  *int64     `json:"employee_id,omitempty"`
	FromDate    *time.Time `json:"from_date,omitempty"`
	ToDate      *time.Time `json:"to_date,omitempty"`
	DayType     *string    `json:"day_type,omitempty"`
}

// BulkTimesheetEntryRequest represents bulk entry creation
type BulkTimesheetEntryRequest struct {
	TimesheetID int64                `json:"timesheet_id" validate:"required"`
	EmployeeID  int64                `json:"employee_id" validate:"required"`
	Entries     []TimesheetEntryData `json:"entries" validate:"required,min=1"`
}

// TimesheetEntryData represents single entry in bulk request
type TimesheetEntryData struct {
	Date          time.Time `json:"date" validate:"required"`
	CheckIn       *string   `json:"check_in,omitempty"`
	CheckOut      *string   `json:"check_out,omitempty"`
	BreakMinutes  int       `json:"break_minutes"`
	WorkedHours   float64   `json:"worked_hours"`
	OvertimeHours float64   `json:"overtime_hours"`
	DayType       string    `json:"day_type" validate:"required"`
	IsRemote      bool      `json:"is_remote"`
	Notes         *string   `json:"notes,omitempty"`
}

// --- Timesheet Correction DTOs ---

// AddTimesheetCorrectionRequest represents request to add correction
type AddTimesheetCorrectionRequest struct {
	EntryID           int64   `json:"entry_id" validate:"required"`
	RequestedCheckIn  *string `json:"requested_check_in,omitempty"`
	RequestedCheckOut *string `json:"requested_check_out,omitempty"`
	RequestedDayType  *string `json:"requested_day_type,omitempty"`
	Reason            string  `json:"reason" validate:"required"`
}

// ApproveCorrectionRequest represents request to approve/reject correction
type ApproveCorrectionRequest struct {
	Approved        bool    `json:"approved"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// CorrectionFilter represents filter for corrections
type CorrectionFilter struct {
	EmployeeID *int64  `json:"employee_id,omitempty"`
	EntryID    *int64  `json:"entry_id,omitempty"`
	Status     *string `json:"status,omitempty"`
	Limit      int     `json:"limit,omitempty"`
	Offset     int     `json:"offset,omitempty"`
}
