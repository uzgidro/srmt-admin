package hrm

import "time"

// VacationType represents types of leave (annual, sick, study, etc.)
type VacationType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`

	Description *string `json:"description,omitempty"`

	DefaultDaysPerYear int  `json:"default_days_per_year"`
	IsPaid             bool `json:"is_paid"`
	RequiresApproval   bool `json:"requires_approval"`
	CanCarryOver       bool `json:"can_carry_over"`
	MaxCarryOverDays   int  `json:"max_carry_over_days"`

	IsActive  bool `json:"is_active"`
	SortOrder int  `json:"sort_order"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// VacationType code constants
const (
	VacationCodeAnnual      = "ANNUAL"
	VacationCodeSick        = "SICK"
	VacationCodeUnpaid      = "UNPAID"
	VacationCodeStudy       = "STUDY"
	VacationCodeMaternity   = "MATERNITY"
	VacationCodePaternity   = "PATERNITY"
	VacationCodeComp        = "COMP"
	VacationCodeMarriage    = "MARRIAGE"
	VacationCodeBereavement = "BEREAVEMENT"
)

// VacationBalance represents employee leave balance per year/type
type VacationBalance struct {
	ID             int64 `json:"id"`
	EmployeeID     int64 `json:"employee_id"`
	VacationTypeID int   `json:"vacation_type_id"`
	Year           int   `json:"year"`

	EntitledDays    float64 `json:"entitled_days"`
	UsedDays        float64 `json:"used_days"`
	CarriedOverDays float64 `json:"carried_over_days"`
	AdjustmentDays  float64 `json:"adjustment_days"`

	// Calculated
	RemainingDays float64 `json:"remaining_days"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	VacationType *VacationType `json:"vacation_type,omitempty"`
	Employee     *Employee     `json:"employee,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// CalculateRemaining calculates remaining vacation days
func (b *VacationBalance) CalculateRemaining() float64 {
	return b.EntitledDays + b.CarriedOverDays + b.AdjustmentDays - b.UsedDays
}

// Vacation represents a leave request
type Vacation struct {
	ID             int64 `json:"id"`
	EmployeeID     int64 `json:"employee_id"`
	VacationTypeID int   `json:"vacation_type_id"`

	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	DaysCount float64   `json:"days_count"`

	Status          string  `json:"status"`
	Reason          *string `json:"reason,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`

	RequestedAt time.Time  `json:"requested_at"`
	ApprovedBy  *int64     `json:"approved_by,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`

	SubstituteEmployeeID *int64 `json:"substitute_employee_id,omitempty"`
	SupportingDocumentID *int64 `json:"supporting_document_id,omitempty"`

	// Enriched
	VacationType       *VacationType `json:"vacation_type,omitempty"`
	Employee           *Employee     `json:"employee,omitempty"`
	SubstituteEmployee *Employee     `json:"substitute_employee,omitempty"`
	ApproverName       *string       `json:"approver_name,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// VacationStatus constants
const (
	VacationStatusPending   = "pending"
	VacationStatusApproved  = "approved"
	VacationStatusRejected  = "rejected"
	VacationStatusCancelled = "cancelled"
)

// VacationCalendarEntry represents a day in vacation calendar
type VacationCalendarEntry struct {
	Date         time.Time `json:"date"`
	EmployeeID   int64     `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	VacationType string    `json:"vacation_type"`
	Status       string    `json:"status"`
}
