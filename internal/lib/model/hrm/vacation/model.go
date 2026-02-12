package vacation

import "time"

type Vacation struct {
	ID              int64     `json:"id"`
	EmployeeID      int64     `json:"employee_id"`
	EmployeeName    string    `json:"employee_name"`
	VacationType    string    `json:"vacation_type"`
	StartDate       string    `json:"start_date"`
	EndDate         string    `json:"end_date"`
	Days            int       `json:"days"`
	Status          string    `json:"status"`
	Reason          *string   `json:"reason,omitempty"`
	RejectionReason *string   `json:"rejection_reason,omitempty"`
	ApprovedBy      *int64    `json:"approved_by,omitempty"`
	ApprovedAt      *string   `json:"approved_at,omitempty"`
	SubstituteID    *int64    `json:"substitute_id,omitempty"`
	SubstituteName  *string   `json:"substitute_name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Balance struct {
	EmployeeID    int64 `json:"employee_id"`
	Year          int   `json:"year"`
	TotalDays     int   `json:"total_days"`
	UsedDays      int   `json:"used_days"`
	PendingDays   int   `json:"pending_days"`
	RemainingDays int   `json:"remaining_days"`
	CarriedOver   int   `json:"carried_over"`
}

type CalendarEntry struct {
	ID           int64  `json:"id"`
	EmployeeID   int64  `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	Department   string `json:"department"`
	VacationType string `json:"vacation_type"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Days         int    `json:"days"`
	Status       string `json:"status"`
}
