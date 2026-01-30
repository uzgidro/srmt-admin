package hrm

import "time"

// --- My Profile DTOs ---

// MyProfileUpdateRequest represents request to update own profile
type MyProfileUpdateRequest struct {
	Phone *string `json:"phone" validate:"omitempty,max=50"`
	Email *string `json:"email" validate:"omitempty,email,max=255"`
}

// --- My Leave Balance DTOs ---

// MyLeaveBalanceResponse represents leave balance for the current employee
type MyLeaveBalanceResponse struct {
	EmployeeID int64                `json:"employee_id"`
	Year       int                  `json:"year"`
	Balances   []LeaveBalanceDetail `json:"balances"`
}

// LeaveBalanceDetail represents individual balance by vacation type
type LeaveBalanceDetail struct {
	VacationTypeID   int     `json:"vacation_type_id"`
	VacationTypeName string  `json:"vacation_type_name"`
	VacationTypeCode string  `json:"vacation_type_code"`
	EntitledDays     float64 `json:"entitled_days"`
	UsedDays         float64 `json:"used_days"`
	CarriedOverDays  float64 `json:"carried_over_days"`
	AdjustmentDays   float64 `json:"adjustment_days"`
	RemainingDays    float64 `json:"remaining_days"`
}

// --- My Vacations DTOs ---

// MyVacationRequest represents request to create vacation request
type MyVacationRequest struct {
	VacationTypeID int64   `json:"type" validate:"required"`
	StartDate      string  `json:"start_date" validate:"required"`
	EndDate        string  `json:"end_date" validate:"required"`
	Reason         *string `json:"reason"`
	SubstituteID   *int64  `json:"substitute_id"`
}

// MyVacationResponse represents vacation request response
type MyVacationResponse struct {
	ID               int64      `json:"id"`
	VacationTypeID   int        `json:"type"`
	VacationTypeName string     `json:"type_name"`
	StartDate        time.Time  `json:"start_date"`
	EndDate          time.Time  `json:"end_date"`
	DaysCount        float64    `json:"days_count"`
	Status           string     `json:"status"`
	Reason           *string    `json:"reason,omitempty"`
	RejectionReason  *string    `json:"rejection_reason,omitempty"`
	RequestedAt      time.Time  `json:"requested_at"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	SubstituteID     *int64     `json:"substitute_id,omitempty"`
	SubstituteName   *string    `json:"substitute_name,omitempty"`
}

// --- My Salary DTOs ---

// MySalaryResponse represents salary info for current employee
type MySalaryResponse struct {
	CurrentSalary   *MySalaryDetail    `json:"current_salary,omitempty"`
	RecentPayslips  []MyPayslipSummary `json:"recent_payslips"`
	SalaryStructure *MySalaryStructure `json:"salary_structure,omitempty"`
}

// MySalaryDetail represents individual salary record
type MySalaryDetail struct {
	ID               int64      `json:"id"`
	Year             int        `json:"year"`
	Month            int        `json:"month"`
	GrossAmount      float64    `json:"gross_amount"`
	NetAmount        float64    `json:"net_amount"`
	TaxAmount        float64    `json:"tax_amount"`
	BonusesAmount    float64    `json:"bonuses_amount"`
	DeductionsAmount float64    `json:"deductions_amount"`
	Status           string     `json:"status"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
}

// MyPayslipSummary represents payslip summary
type MyPayslipSummary struct {
	ID        int64      `json:"id"`
	Year      int        `json:"year"`
	Month     int        `json:"month"`
	NetAmount float64    `json:"net_amount"`
	Status    string     `json:"status"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
}

// MySalaryStructure represents current salary structure
type MySalaryStructure struct {
	BaseSalary    float64 `json:"base_salary"`
	Currency      string  `json:"currency"`
	PayFrequency  string  `json:"pay_frequency"`
	EffectiveFrom string  `json:"effective_from"`
}

// --- My Training DTOs ---

// MyTrainingResponse represents training data for current employee
type MyTrainingResponse struct {
	Enrollments  []MyTrainingEnrollment `json:"enrollments"`
	Certificates []MyCertificate        `json:"certificates"`
}

// MyTrainingEnrollment represents training enrollment
type MyTrainingEnrollment struct {
	ID                int64      `json:"id"`
	TrainingID        int64      `json:"training_id"`
	TrainingTitle     string     `json:"training_title"`
	TrainingType      string     `json:"training_type"`
	Status            string     `json:"status"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	Location          *string    `json:"location,omitempty"`
	IsMandatory       bool       `json:"is_mandatory"`
	AttendancePercent *int       `json:"attendance_percent,omitempty"`
	Score             *float64   `json:"score,omitempty"`
	Passed            *bool      `json:"passed,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
}

// MyCertificate represents certificate
type MyCertificate struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Issuer            string     `json:"issuer"`
	CertificateNumber *string    `json:"certificate_number,omitempty"`
	IssuedDate        time.Time  `json:"issued_date"`
	ExpiryDate        *time.Time `json:"expiry_date,omitempty"`
	IsExpired         bool       `json:"is_expired"`
	IsVerified        bool       `json:"is_verified"`
}

// --- My Competencies DTOs ---

// MyCompetenciesResponse represents competencies for current employee
type MyCompetenciesResponse struct {
	Scores      []MyCompetencyScore `json:"scores"`
	Assessments []MyAssessment      `json:"assessments"`
}

// MyCompetencyScore represents competency score
type MyCompetencyScore struct {
	CompetencyID   int64     `json:"competency_id"`
	CompetencyName string    `json:"competency_name"`
	CategoryName   string    `json:"category_name"`
	Score          int       `json:"score"`
	MaxScore       int       `json:"max_score"`
	LevelName      *string   `json:"level_name,omitempty"`
	AssessmentDate time.Time `json:"assessment_date"`
}

// MyAssessment represents assessment
type MyAssessment struct {
	ID             int64      `json:"id"`
	AssessmentType string     `json:"assessment_type"`
	Status         string     `json:"status"`
	ScheduledDate  *time.Time `json:"scheduled_date,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// --- My Notifications DTOs ---

// MyNotificationResponse represents notification
type MyNotificationResponse struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	Category    string     `json:"category"`
	Priority    string     `json:"priority"`
	IsRead      bool       `json:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	ActionURL   *string    `json:"action_url,omitempty"`
	ActionLabel *string    `json:"action_label,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// --- My Tasks DTOs ---

// MyTaskResponse represents task
type MyTaskResponse struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	TaskType    string     `json:"task_type"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	EntityType  *string    `json:"entity_type,omitempty"`
	EntityID    *int64     `json:"entity_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// --- My Documents DTOs ---

// MyDocumentResponse represents document
type MyDocumentResponse struct {
	ID           int64      `json:"id"`
	DocumentType string     `json:"document_type"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	IssuedDate   *time.Time `json:"issued_date,omitempty"`
	ExpiryDate   *time.Time `json:"expiry_date,omitempty"`
	FileID       *int64     `json:"file_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
