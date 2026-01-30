package hrm

import (
	"time"

	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
)

// Vacancy represents a job opening
type Vacancy struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	PositionID     *int64 `json:"position_id,omitempty"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	OrganizationID *int64 `json:"organization_id,omitempty"`

	// Details
	Description      *string `json:"description,omitempty"`
	Requirements     *string `json:"requirements,omitempty"`
	Responsibilities *string `json:"responsibilities,omitempty"`
	Benefits         *string `json:"benefits,omitempty"`

	// Employment terms
	EmploymentType  string  `json:"employment_type"`
	WorkFormat      string  `json:"work_format"`
	ExperienceLevel *string `json:"experience_level,omitempty"`

	// Salary
	SalaryMin     *float64 `json:"salary_min,omitempty"`
	SalaryMax     *float64 `json:"salary_max,omitempty"`
	Currency      string   `json:"currency"`
	SalaryVisible bool     `json:"salary_visible"`

	// Status
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Headcount   int    `json:"headcount"`
	FilledCount int    `json:"filled_count"`

	// Dates
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`

	// Responsible
	HiringManagerID *int64 `json:"hiring_manager_id,omitempty"`
	RecruiterID     *int64 `json:"recruiter_id,omitempty"`

	// Enriched
	Position      *position.Model     `json:"position,omitempty"`
	Department    *department.Model   `json:"department,omitempty"`
	Organization  *organization.Model `json:"organization,omitempty"`
	HiringManager *Employee           `json:"hiring_manager,omitempty"`
	Recruiter     *Employee           `json:"recruiter,omitempty"`

	// Stats
	CandidateCount int `json:"candidate_count,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// VacancyStatus constants
const (
	VacancyStatusDraft  = "draft"
	VacancyStatusOpen   = "open"
	VacancyStatusPaused = "paused"
	VacancyStatusClosed = "closed"
	VacancyStatusFilled = "filled"
)

// WorkFormat constants
const (
	WorkFormatOffice = "office"
	WorkFormatRemote = "remote"
	WorkFormatHybrid = "hybrid"
)

// ExperienceLevel constants
const (
	ExperienceLevelJunior = "junior"
	ExperienceLevelMiddle = "middle"
	ExperienceLevelSenior = "senior"
	ExperienceLevelLead   = "lead"
)

// VacancyPriority constants
const (
	VacancyPriorityLow    = "low"
	VacancyPriorityNormal = "normal"
	VacancyPriorityHigh   = "high"
	VacancyPriorityUrgent = "urgent"
)

// Candidate represents a job applicant
type Candidate struct {
	ID        int64 `json:"id"`
	VacancyID int64 `json:"vacancy_id"`

	// Personal info
	FirstName  string  `json:"first_name"`
	LastName   string  `json:"last_name"`
	MiddleName *string `json:"middle_name,omitempty"`
	Email      *string `json:"email,omitempty"`
	Phone      *string `json:"phone,omitempty"`

	// Professional info
	CurrentPosition *string  `json:"current_position,omitempty"`
	CurrentCompany  *string  `json:"current_company,omitempty"`
	ExperienceYears *int     `json:"experience_years,omitempty"`
	ExpectedSalary  *float64 `json:"expected_salary,omitempty"`
	Currency        string   `json:"currency"`

	// Documents
	ResumeFileID *int64  `json:"resume_file_id,omitempty"`
	CoverLetter  *string `json:"cover_letter,omitempty"`

	// Source
	Source             *string `json:"source,omitempty"`
	ReferrerEmployeeID *int64  `json:"referrer_employee_id,omitempty"`

	// Status
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejection_reason,omitempty"`

	// Rating
	Rating *int    `json:"rating,omitempty"`
	Notes  *string `json:"notes,omitempty"`

	// Enriched
	Vacancy          *Vacancy    `json:"vacancy,omitempty"`
	ReferrerEmployee *Employee   `json:"referrer_employee,omitempty"`
	ResumeURL        *string     `json:"resume_url,omitempty"`
	Interviews       []Interview `json:"interviews,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// CandidateStatus constants
const (
	CandidateStatusNew       = "new"
	CandidateStatusScreening = "screening"
	CandidateStatusInterview = "interview"
	CandidateStatusOffer     = "offer"
	CandidateStatusHired     = "hired"
	CandidateStatusRejected  = "rejected"
	CandidateStatusWithdrawn = "withdrawn"
)

// CandidateSource constants
const (
	CandidateSourceHH       = "hh.ru"
	CandidateSourceLinkedIn = "linkedin"
	CandidateSourceReferral = "referral"
	CandidateSourceWebsite  = "website"
	CandidateSourceAgency   = "agency"
	CandidateSourceDirect   = "direct"
)

// Interview represents an interview record
type Interview struct {
	ID          int64 `json:"id"`
	CandidateID int64 `json:"candidate_id"`
	VacancyID   int64 `json:"vacancy_id"`

	// Details
	InterviewType   string    `json:"interview_type"`
	ScheduledAt     time.Time `json:"scheduled_at"`
	DurationMinutes int       `json:"duration_minutes"`
	Location        *string   `json:"location,omitempty"`

	// Interviewers
	InterviewerIDs []int64 `json:"interviewer_ids"`

	// Results
	Status         string     `json:"status"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	OverallRating  *int       `json:"overall_rating,omitempty"`
	Feedback       *string    `json:"feedback,omitempty"`
	Recommendation *string    `json:"recommendation,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Candidate    *Candidate `json:"candidate,omitempty"`
	Vacancy      *Vacancy   `json:"vacancy,omitempty"`
	Interviewers []Employee `json:"interviewers,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// InterviewType constants
const (
	InterviewTypePhone     = "phone"
	InterviewTypeVideo     = "video"
	InterviewTypeOnsite    = "onsite"
	InterviewTypeTechnical = "technical"
	InterviewTypeHR        = "hr"
	InterviewTypeFinal     = "final"
)

// InterviewStatus constants
const (
	InterviewStatusScheduled = "scheduled"
	InterviewStatusCompleted = "completed"
	InterviewStatusCancelled = "cancelled"
	InterviewStatusNoShow    = "no_show"
)

// InterviewRecommendation constants
const (
	InterviewRecommendationHire   = "hire"
	InterviewRecommendationMaybe  = "maybe"
	InterviewRecommendationReject = "reject"
)

// RecruitingStats represents recruiting metrics
type RecruitingStats struct {
	OpenVacancies       int     `json:"open_vacancies"`
	TotalCandidates     int     `json:"total_candidates"`
	NewCandidates       int     `json:"new_candidates"`
	ScheduledInterviews int     `json:"scheduled_interviews"`
	OffersExtended      int     `json:"offers_extended"`
	HiredThisMonth      int     `json:"hired_this_month"`
	AverageTimeToHire   float64 `json:"average_time_to_hire_days"`
}
