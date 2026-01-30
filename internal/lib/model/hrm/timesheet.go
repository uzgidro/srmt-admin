package hrm

import "time"

// Holiday represents a public holiday
type Holiday struct {
	ID   int       `json:"id"`
	Name string    `json:"name"`
	Date time.Time `json:"date"`
	Year int       `json:"year"`

	IsWorkingDay bool `json:"is_working_day"` // For special working Saturdays

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// Timesheet represents monthly timesheet summary
type Timesheet struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`
	Year       int   `json:"year"`
	Month      int   `json:"month"`

	// Summary
	TotalWorkDays   int     `json:"total_work_days"`
	TotalWorkedDays int     `json:"total_worked_days"`
	TotalHours      float64 `json:"total_hours"`
	OvertimeHours   float64 `json:"overtime_hours"`
	SickDays        int     `json:"sick_days"`
	VacationDays    int     `json:"vacation_days"`
	AbsenceDays     int     `json:"absence_days"`

	// Status
	Status          string     `json:"status"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	ApprovedBy      *int64     `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee     *Employee        `json:"employee,omitempty"`
	Entries      []TimesheetEntry `json:"entries,omitempty"`
	ApproverName *string          `json:"approver_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// TimesheetStatus constants
const (
	TimesheetStatusDraft     = "draft"
	TimesheetStatusSubmitted = "submitted"
	TimesheetStatusApproved  = "approved"
	TimesheetStatusRejected  = "rejected"
)

// TimesheetEntry represents daily work entry
type TimesheetEntry struct {
	ID          int64     `json:"id"`
	TimesheetID int64     `json:"timesheet_id"`
	EmployeeID  int64     `json:"employee_id"`
	Date        time.Time `json:"date"`

	// Time tracking
	CheckIn       *string `json:"check_in,omitempty"`  // TIME as string "HH:MM"
	CheckOut      *string `json:"check_out,omitempty"` // TIME as string "HH:MM"
	BreakMinutes  int     `json:"break_minutes"`
	WorkedHours   float64 `json:"worked_hours"`
	OvertimeHours float64 `json:"overtime_hours"`

	// Day type
	DayType  string `json:"day_type"`
	IsRemote bool   `json:"is_remote"`

	Notes *string `json:"notes,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// DayType constants
const (
	DayTypeWork     = "work"
	DayTypeWeekend  = "weekend"
	DayTypeHoliday  = "holiday"
	DayTypeVacation = "vacation"
	DayTypeSick     = "sick"
	DayTypeAbsence  = "absence"
	DayTypeRemote   = "remote"
)

// TimesheetCorrection represents correction request
type TimesheetCorrection struct {
	ID         int64 `json:"id"`
	EntryID    int64 `json:"entry_id"`
	EmployeeID int64 `json:"employee_id"`

	// Original values
	OriginalCheckIn  *string `json:"original_check_in,omitempty"`
	OriginalCheckOut *string `json:"original_check_out,omitempty"`
	OriginalDayType  *string `json:"original_day_type,omitempty"`

	// Requested values
	RequestedCheckIn  *string `json:"requested_check_in,omitempty"`
	RequestedCheckOut *string `json:"requested_check_out,omitempty"`
	RequestedDayType  *string `json:"requested_day_type,omitempty"`

	Reason string `json:"reason"`

	// Approval
	Status          string     `json:"status"`
	ApprovedBy      *int64     `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`

	// Enriched
	Entry        *TimesheetEntry `json:"entry,omitempty"`
	Employee     *Employee       `json:"employee,omitempty"`
	ApproverName *string         `json:"approver_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// CorrectionStatus constants
const (
	CorrectionStatusPending  = "pending"
	CorrectionStatusApproved = "approved"
	CorrectionStatusRejected = "rejected"
)

// TimesheetSummary represents aggregated timesheet data
type TimesheetSummary struct {
	Year           int     `json:"year"`
	Month          int     `json:"month"`
	TotalEmployees int     `json:"total_employees"`
	TotalHours     float64 `json:"total_hours"`
	OvertimeHours  float64 `json:"overtime_hours"`
	SickDays       int     `json:"sick_days"`
	VacationDays   int     `json:"vacation_days"`
	AbsenceDays    int     `json:"absence_days"`
}
