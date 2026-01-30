package hrm

import "time"

// Training represents a training course/program
type Training struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`

	Description *string `json:"description,omitempty"`

	// Details
	TrainingType string  `json:"training_type"`
	Category     *string `json:"category,omitempty"`
	Provider     *string `json:"provider,omitempty"`
	Instructor   *string `json:"instructor,omitempty"`

	// Schedule
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	DurationHours *int       `json:"duration_hours,omitempty"`
	Location      *string    `json:"location,omitempty"`

	// Capacity
	MaxParticipants *int `json:"max_participants,omitempty"`
	MinParticipants *int `json:"min_participants,omitempty"`

	// Cost
	CostPerParticipant *float64 `json:"cost_per_participant,omitempty"`
	Currency           string   `json:"currency"`

	// Status
	Status      string `json:"status"`
	IsMandatory bool   `json:"is_mandatory"`

	// Materials
	MaterialsFileID *int64 `json:"materials_file_id,omitempty"`

	// Responsible
	OrganizerID *int64 `json:"organizer_id,omitempty"`

	// Enriched
	Organizer        *Employee             `json:"organizer,omitempty"`
	MaterialsURL     *string               `json:"materials_url,omitempty"`
	ParticipantCount int                   `json:"participant_count,omitempty"`
	Participants     []TrainingParticipant `json:"participants,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// TrainingType constants
const (
	TrainingTypeInternal      = "internal"
	TrainingTypeExternal      = "external"
	TrainingTypeOnline        = "online"
	TrainingTypeWorkshop      = "workshop"
	TrainingTypeCertification = "certification"
)

// TrainingCategory constants
const (
	TrainingCategoryTechnical  = "technical"
	TrainingCategorySoftSkills = "soft_skills"
	TrainingCategoryCompliance = "compliance"
	TrainingCategorySafety     = "safety"
	TrainingCategoryLeadership = "leadership"
)

// TrainingStatus constants
const (
	TrainingStatusPlanned      = "planned"
	TrainingStatusRegistration = "registration"
	TrainingStatusInProgress   = "in_progress"
	TrainingStatusCompleted    = "completed"
	TrainingStatusCancelled    = "cancelled"
)

// TrainingParticipant represents employee enrollment
type TrainingParticipant struct {
	ID         int64 `json:"id"`
	TrainingID int64 `json:"training_id"`
	EmployeeID int64 `json:"employee_id"`

	// Enrollment
	EnrolledAt time.Time `json:"enrolled_at"`
	EnrolledBy *int64    `json:"enrolled_by,omitempty"`

	// Status
	Status string `json:"status"`

	// Results
	AttendancePercent *int       `json:"attendance_percent,omitempty"`
	Score             *float64   `json:"score,omitempty"`
	Passed            *bool      `json:"passed,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`

	// Feedback
	FeedbackRating *int    `json:"feedback_rating,omitempty"`
	FeedbackText   *string `json:"feedback_text,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Training *Training `json:"training,omitempty"`
	Employee *Employee `json:"employee,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// ParticipantStatus constants
const (
	ParticipantStatusEnrolled  = "enrolled"
	ParticipantStatusAttended  = "attended"
	ParticipantStatusCompleted = "completed"
	ParticipantStatusNoShow    = "no_show"
	ParticipantStatusCancelled = "cancelled"
)

// Certificate represents employee certificate
type Certificate struct {
	ID         int64  `json:"id"`
	EmployeeID int64  `json:"employee_id"`
	TrainingID *int64 `json:"training_id,omitempty"`

	// Certificate info
	Name              string  `json:"name"`
	Issuer            string  `json:"issuer"`
	CertificateNumber *string `json:"certificate_number,omitempty"`

	// Dates
	IssuedDate time.Time  `json:"issued_date"`
	ExpiryDate *time.Time `json:"expiry_date,omitempty"`

	// File
	FileID *int64 `json:"file_id,omitempty"`

	// Verification
	IsVerified bool       `json:"is_verified"`
	VerifiedBy *int64     `json:"verified_by,omitempty"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee *Employee `json:"employee,omitempty"`
	Training *Training `json:"training,omitempty"`
	FileURL  *string   `json:"file_url,omitempty"`

	// Calculated
	IsExpired bool `json:"is_expired,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// DevelopmentPlan represents Individual Development Plan (IDP)
type DevelopmentPlan struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	// Plan period
	Title     string    `json:"title"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`

	// Status
	Status string `json:"status"`

	// Review
	ManagerID   *int64     `json:"manager_id,omitempty"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	OverallProgress int     `json:"overall_progress"`
	Notes           *string `json:"notes,omitempty"`

	// Enriched
	Employee *Employee         `json:"employee,omitempty"`
	Manager  *Employee         `json:"manager,omitempty"`
	Goals    []DevelopmentGoal `json:"goals,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// DevelopmentPlanStatus constants
const (
	DevelopmentPlanStatusDraft     = "draft"
	DevelopmentPlanStatusActive    = "active"
	DevelopmentPlanStatusCompleted = "completed"
	DevelopmentPlanStatusCancelled = "cancelled"
)

// DevelopmentGoal represents goals within IDP
type DevelopmentGoal struct {
	ID         int64 `json:"id"`
	PlanID     int64 `json:"plan_id"`
	EmployeeID int64 `json:"employee_id"`

	// Goal details
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`

	// Target
	TargetDate *time.Time `json:"target_date,omitempty"`
	Priority   string     `json:"priority"`

	// Progress
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Related training
	TrainingID *int64 `json:"training_id,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Training *Training `json:"training,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// DevelopmentGoalCategory constants
const (
	DevelopmentGoalCategorySkill         = "skill"
	DevelopmentGoalCategoryCertification = "certification"
	DevelopmentGoalCategoryProject       = "project"
	DevelopmentGoalCategoryRole          = "role"
)

// DevelopmentGoalStatus constants
const (
	DevelopmentGoalStatusNotStarted = "not_started"
	DevelopmentGoalStatusInProgress = "in_progress"
	DevelopmentGoalStatusCompleted  = "completed"
	DevelopmentGoalStatusCancelled  = "cancelled"
)

// Priority constants
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

// TrainingStats represents training metrics
type TrainingStats struct {
	TotalTrainings       int     `json:"total_trainings"`
	ActiveTrainings      int     `json:"active_trainings"`
	UpcomingTrainings    int     `json:"upcoming_trainings"`
	TotalParticipants    int     `json:"total_participants"`
	CompletionRate       float64 `json:"completion_rate_percent"`
	AverageFeedback      float64 `json:"average_feedback_rating"`
	ExpiringCertificates int     `json:"expiring_certificates_30d"`
}
