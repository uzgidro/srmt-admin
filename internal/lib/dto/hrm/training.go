package hrm

import "time"

// --- Training DTOs ---

// AddTrainingRequest represents request to create training
type AddTrainingRequest struct {
	Title              string     `json:"title" validate:"required"`
	Description        *string    `json:"description,omitempty"`
	TrainingType       string     `json:"training_type" validate:"required,oneof=internal external online workshop certification"`
	Category           *string    `json:"category,omitempty"`
	Provider           *string    `json:"provider,omitempty"`
	Instructor         *string    `json:"instructor,omitempty"`
	StartDate          *time.Time `json:"start_date,omitempty"`
	EndDate            *time.Time `json:"end_date,omitempty"`
	DurationHours      *int       `json:"duration_hours,omitempty"`
	Location           *string    `json:"location,omitempty"`
	MaxParticipants    *int       `json:"max_participants,omitempty"`
	MinParticipants    *int       `json:"min_participants,omitempty"`
	CostPerParticipant *float64   `json:"cost_per_participant,omitempty"`
	Currency           string     `json:"currency" validate:"required,len=3"`
	IsMandatory        bool       `json:"is_mandatory"`
	MaterialsFileID    *int64     `json:"materials_file_id,omitempty"`
	OrganizerID        *int64     `json:"organizer_id,omitempty"`
}

// EditTrainingRequest represents request to update training
type EditTrainingRequest struct {
	Title              *string    `json:"title,omitempty"`
	Description        *string    `json:"description,omitempty"`
	TrainingType       *string    `json:"training_type,omitempty"`
	Category           *string    `json:"category,omitempty"`
	Provider           *string    `json:"provider,omitempty"`
	Instructor         *string    `json:"instructor,omitempty"`
	StartDate          *time.Time `json:"start_date,omitempty"`
	EndDate            *time.Time `json:"end_date,omitempty"`
	DurationHours      *int       `json:"duration_hours,omitempty"`
	Location           *string    `json:"location,omitempty"`
	MaxParticipants    *int       `json:"max_participants,omitempty"`
	MinParticipants    *int       `json:"min_participants,omitempty"`
	CostPerParticipant *float64   `json:"cost_per_participant,omitempty"`
	Currency           *string    `json:"currency,omitempty"`
	Status             *string    `json:"status,omitempty"`
	IsMandatory        *bool      `json:"is_mandatory,omitempty"`
	MaterialsFileID    *int64     `json:"materials_file_id,omitempty"`
	OrganizerID        *int64     `json:"organizer_id,omitempty"`
}

// TrainingFilter represents filter for trainings
type TrainingFilter struct {
	TrainingType *string    `json:"training_type,omitempty"`
	Category     *string    `json:"category,omitempty"`
	Status       *string    `json:"status,omitempty"`
	IsMandatory  *bool      `json:"is_mandatory,omitempty"`
	FromDate     *time.Time `json:"from_date,omitempty"`
	ToDate       *time.Time `json:"to_date,omitempty"`
	OrganizerID  *int64     `json:"organizer_id,omitempty"`
	Search       *string    `json:"search,omitempty"` // Title search
	Limit        int        `json:"limit,omitempty"`
	Offset       int        `json:"offset,omitempty"`
}

// --- Training Participant DTOs ---

// EnrollParticipantRequest represents request to enroll participant
type EnrollParticipantRequest struct {
	TrainingID int64 `json:"training_id" validate:"required"`
	EmployeeID int64 `json:"employee_id" validate:"required"`
}

// BulkEnrollRequest represents request to enroll multiple participants
type BulkEnrollRequest struct {
	TrainingID  int64   `json:"training_id" validate:"required"`
	EmployeeIDs []int64 `json:"employee_ids" validate:"required,min=1"`
}

// UpdateParticipantRequest represents request to update participant
type UpdateParticipantRequest struct {
	Status            *string  `json:"status,omitempty"`
	AttendancePercent *int     `json:"attendance_percent,omitempty"`
	Score             *float64 `json:"score,omitempty"`
	Passed            *bool    `json:"passed,omitempty"`
	FeedbackRating    *int     `json:"feedback_rating,omitempty"`
	FeedbackText      *string  `json:"feedback_text,omitempty"`
	Notes             *string  `json:"notes,omitempty"`
}

// CompleteTrainingRequest represents request to mark training as completed
type CompleteTrainingRequest struct {
	Score  *float64 `json:"score,omitempty"`
	Passed bool     `json:"passed"`
}

// ParticipantFilter represents filter for participants
type ParticipantFilter struct {
	TrainingID *int64  `json:"training_id,omitempty"`
	EmployeeID *int64  `json:"employee_id,omitempty"`
	Status     *string `json:"status,omitempty"`
	Passed     *bool   `json:"passed,omitempty"`
}

// --- Certificate DTOs ---

// AddCertificateRequest represents request to add certificate
type AddCertificateRequest struct {
	EmployeeID        int64      `json:"employee_id" validate:"required"`
	TrainingID        *int64     `json:"training_id,omitempty"`
	Name              string     `json:"name" validate:"required"`
	Issuer            string     `json:"issuer" validate:"required"`
	CertificateNumber *string    `json:"certificate_number,omitempty"`
	IssuedDate        time.Time  `json:"issued_date" validate:"required"`
	ExpiryDate        *time.Time `json:"expiry_date,omitempty"`
	FileID            *int64     `json:"file_id,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
}

// EditCertificateRequest represents request to update certificate
type EditCertificateRequest struct {
	Name              *string    `json:"name,omitempty"`
	Issuer            *string    `json:"issuer,omitempty"`
	CertificateNumber *string    `json:"certificate_number,omitempty"`
	IssuedDate        *time.Time `json:"issued_date,omitempty"`
	ExpiryDate        *time.Time `json:"expiry_date,omitempty"`
	FileID            *int64     `json:"file_id,omitempty"`
	IsVerified        *bool      `json:"is_verified,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
}

// CertificateFilter represents filter for certificates
type CertificateFilter struct {
	EmployeeID   *int64  `json:"employee_id,omitempty"`
	TrainingID   *int64  `json:"training_id,omitempty"`
	IsVerified   *bool   `json:"is_verified,omitempty"`
	ExpiringDays *int    `json:"expiring_days,omitempty"` // Expiring within N days
	Expired      *bool   `json:"expired,omitempty"`
	Search       *string `json:"search,omitempty"` // Name, issuer search
	Limit        int     `json:"limit,omitempty"`
	Offset       int     `json:"offset,omitempty"`
}

// --- Development Plan DTOs ---

// AddDevelopmentPlanRequest represents request to create IDP
type AddDevelopmentPlanRequest struct {
	EmployeeID int64     `json:"employee_id" validate:"required"`
	Title      string    `json:"title" validate:"required"`
	StartDate  time.Time `json:"start_date" validate:"required"`
	EndDate    time.Time `json:"end_date" validate:"required"`
	ManagerID  *int64    `json:"manager_id,omitempty"`
	Notes      *string   `json:"notes,omitempty"`
}

// EditDevelopmentPlanRequest represents request to update IDP
type EditDevelopmentPlanRequest struct {
	Title           *string    `json:"title,omitempty"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	Status          *string    `json:"status,omitempty"`
	ManagerID       *int64     `json:"manager_id,omitempty"`
	OverallProgress *int       `json:"overall_progress,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
}

// DevelopmentPlanFilter represents filter for development plans
type DevelopmentPlanFilter struct {
	EmployeeID *int64  `json:"employee_id,omitempty"`
	ManagerID  *int64  `json:"manager_id,omitempty"`
	Status     *string `json:"status,omitempty"`
	Limit      int     `json:"limit,omitempty"`
	Offset     int     `json:"offset,omitempty"`
}

// --- Development Goal DTOs ---

// AddDevelopmentGoalRequest represents request to add goal to IDP
type AddDevelopmentGoalRequest struct {
	PlanID      int64      `json:"plan_id" validate:"required"`
	EmployeeID  int64      `json:"employee_id" validate:"required"`
	Title       string     `json:"title" validate:"required"`
	Description *string    `json:"description,omitempty"`
	Category    *string    `json:"category,omitempty"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	Priority    string     `json:"priority" validate:"omitempty,oneof=low normal high"`
	TrainingID  *int64     `json:"training_id,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

// EditDevelopmentGoalRequest represents request to update goal
type EditDevelopmentGoalRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Category    *string    `json:"category,omitempty"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	Priority    *string    `json:"priority,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Progress    *int       `json:"progress,omitempty"`
	TrainingID  *int64     `json:"training_id,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

// DevelopmentGoalFilter represents filter for development goals
type DevelopmentGoalFilter struct {
	PlanID     *int64  `json:"plan_id,omitempty"`
	EmployeeID *int64  `json:"employee_id,omitempty"`
	Status     *string `json:"status,omitempty"`
	Category   *string `json:"category,omitempty"`
	Priority   *string `json:"priority,omitempty"`
}
