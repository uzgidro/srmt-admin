package timesheet

import "time"

// Day represents one day in an employee's timesheet
type Day struct {
	ID          int64    `json:"id,omitempty"`
	EmployeeID  int64    `json:"employee_id"`
	Date        string   `json:"date"`
	Status      string   `json:"status"`
	CheckIn     *string  `json:"check_in,omitempty"`
	CheckOut    *string  `json:"check_out,omitempty"`
	HoursWorked *float64 `json:"hours_worked,omitempty"`
	Overtime    *float64 `json:"overtime,omitempty"`
	IsWeekend   bool     `json:"is_weekend"`
	IsHoliday   bool     `json:"is_holiday"`
	Note        *string  `json:"note,omitempty"`
}

// Summary is the monthly aggregation for an employee's timesheet
type Summary struct {
	TotalWorkDays    int     `json:"total_work_days"`
	PresentDays      int     `json:"present_days"`
	AbsentDays       int     `json:"absent_days"`
	VacationDays     int     `json:"vacation_days"`
	SickDays         int     `json:"sick_days"`
	BusinessTripDays int     `json:"business_trip_days"`
	RemoteDays       int     `json:"remote_days"`
	TotalHours       float64 `json:"total_hours"`
	OvertimeHours    float64 `json:"overtime_hours"`
	LateArrivals     int     `json:"late_arrivals"`
	EarlyDepartures  int     `json:"early_departures"`
}

// EmployeeTimesheet is one employee's timesheet for a month (days + summary)
type EmployeeTimesheet struct {
	EmployeeID   int64   `json:"employee_id"`
	EmployeeName string  `json:"employee_name"`
	Department   string  `json:"department"`
	Position     string  `json:"position"`
	TabNumber    string  `json:"tab_number"`
	Days         []Day   `json:"days"`
	Summary      Summary `json:"summary"`
}

// Holiday represents a holiday day
type Holiday struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Date        string    `json:"date"`
	Type        string    `json:"type"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Correction represents a timesheet correction request
type Correction struct {
	ID               int64     `json:"id"`
	EmployeeID       int64     `json:"employee_id"`
	EmployeeName     string    `json:"employee_name"`
	Date             string    `json:"date"`
	OriginalStatus   *string   `json:"original_status,omitempty"`
	NewStatus        string    `json:"new_status"`
	OriginalCheckIn  *string   `json:"original_check_in,omitempty"`
	NewCheckIn       *string   `json:"new_check_in,omitempty"`
	OriginalCheckOut *string   `json:"original_check_out,omitempty"`
	NewCheckOut      *string   `json:"new_check_out,omitempty"`
	Reason           string    `json:"reason"`
	Status           string    `json:"status"`
	RequestedBy      int64     `json:"requested_by"`
	ApprovedBy       *int64    `json:"approved_by,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EmployeeInfo holds basic employee info for timesheet generation
type EmployeeInfo struct {
	EmployeeID int64
	Name       string
	Department string
	Position   string
	TabNumber  string
}
