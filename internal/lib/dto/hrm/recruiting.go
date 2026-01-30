package hrm

import "time"

// --- Vacancy DTOs ---

// AddVacancyRequest represents request to create vacancy
type AddVacancyRequest struct {
	Title            string     `json:"title" validate:"required"`
	PositionID       *int64     `json:"position_id,omitempty"`
	DepartmentID     *int64     `json:"department_id,omitempty"`
	OrganizationID   *int64     `json:"organization_id,omitempty"`
	Description      *string    `json:"description,omitempty"`
	Requirements     *string    `json:"requirements,omitempty"`
	Responsibilities *string    `json:"responsibilities,omitempty"`
	Benefits         *string    `json:"benefits,omitempty"`
	EmploymentType   string     `json:"employment_type" validate:"required,oneof=full_time part_time contract intern"`
	WorkFormat       string     `json:"work_format" validate:"required,oneof=office remote hybrid"`
	ExperienceLevel  *string    `json:"experience_level,omitempty"`
	SalaryMin        *float64   `json:"salary_min,omitempty"`
	SalaryMax        *float64   `json:"salary_max,omitempty"`
	Currency         string     `json:"currency" validate:"required,len=3"`
	SalaryVisible    bool       `json:"salary_visible"`
	Priority         string     `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	Headcount        int        `json:"headcount" validate:"required,min=1"`
	Deadline         *time.Time `json:"deadline,omitempty"`
	HiringManagerID  *int64     `json:"hiring_manager_id,omitempty"`
	RecruiterID      *int64     `json:"recruiter_id,omitempty"`
}

// EditVacancyRequest represents request to update vacancy
type EditVacancyRequest struct {
	Title            *string    `json:"title,omitempty"`
	PositionID       *int64     `json:"position_id,omitempty"`
	DepartmentID     *int64     `json:"department_id,omitempty"`
	OrganizationID   *int64     `json:"organization_id,omitempty"`
	Description      *string    `json:"description,omitempty"`
	Requirements     *string    `json:"requirements,omitempty"`
	Responsibilities *string    `json:"responsibilities,omitempty"`
	Benefits         *string    `json:"benefits,omitempty"`
	EmploymentType   *string    `json:"employment_type,omitempty"`
	WorkFormat       *string    `json:"work_format,omitempty"`
	ExperienceLevel  *string    `json:"experience_level,omitempty"`
	SalaryMin        *float64   `json:"salary_min,omitempty"`
	SalaryMax        *float64   `json:"salary_max,omitempty"`
	Currency         *string    `json:"currency,omitempty"`
	SalaryVisible    *bool      `json:"salary_visible,omitempty"`
	Status           *string    `json:"status,omitempty"`
	Priority         *string    `json:"priority,omitempty"`
	Headcount        *int       `json:"headcount,omitempty"`
	Deadline         *time.Time `json:"deadline,omitempty"`
	HiringManagerID  *int64     `json:"hiring_manager_id,omitempty"`
	RecruiterID      *int64     `json:"recruiter_id,omitempty"`
}

// PublishVacancyRequest represents request to publish vacancy
type PublishVacancyRequest struct {
	Publish bool `json:"publish"`
}

// VacancyFilter represents filter for vacancies
type VacancyFilter struct {
	Status          *string `json:"status,omitempty"`
	DepartmentID    *int64  `json:"department_id,omitempty"`
	OrganizationID  *int64  `json:"organization_id,omitempty"`
	PositionID      *int64  `json:"position_id,omitempty"`
	EmploymentType  *string `json:"employment_type,omitempty"`
	WorkFormat      *string `json:"work_format,omitempty"`
	Priority        *string `json:"priority,omitempty"`
	HiringManagerID *int64  `json:"hiring_manager_id,omitempty"`
	RecruiterID     *int64  `json:"recruiter_id,omitempty"`
	Search          *string `json:"search,omitempty"` // Title search
	Limit           int     `json:"limit,omitempty"`
	Offset          int     `json:"offset,omitempty"`
}

// --- Candidate DTOs ---

// AddCandidateRequest represents request to add candidate
type AddCandidateRequest struct {
	VacancyID          int64    `json:"vacancy_id" validate:"required"`
	FirstName          string   `json:"first_name" validate:"required"`
	LastName           string   `json:"last_name" validate:"required"`
	MiddleName         *string  `json:"middle_name,omitempty"`
	Email              *string  `json:"email,omitempty" validate:"omitempty,email"`
	Phone              *string  `json:"phone,omitempty"`
	CurrentPosition    *string  `json:"current_position,omitempty"`
	CurrentCompany     *string  `json:"current_company,omitempty"`
	ExperienceYears    *int     `json:"experience_years,omitempty"`
	ExpectedSalary     *float64 `json:"expected_salary,omitempty"`
	Currency           string   `json:"currency" validate:"required,len=3"`
	ResumeFileID       *int64   `json:"resume_file_id,omitempty"`
	CoverLetter        *string  `json:"cover_letter,omitempty"`
	Source             *string  `json:"source,omitempty"`
	ReferrerEmployeeID *int64   `json:"referrer_employee_id,omitempty"`
	Notes              *string  `json:"notes,omitempty"`
}

// EditCandidateRequest represents request to update candidate
type EditCandidateRequest struct {
	VacancyID          *int64   `json:"vacancy_id,omitempty"`
	FirstName          *string  `json:"first_name,omitempty"`
	LastName           *string  `json:"last_name,omitempty"`
	MiddleName         *string  `json:"middle_name,omitempty"`
	Email              *string  `json:"email,omitempty"`
	Phone              *string  `json:"phone,omitempty"`
	CurrentPosition    *string  `json:"current_position,omitempty"`
	CurrentCompany     *string  `json:"current_company,omitempty"`
	ExperienceYears    *int     `json:"experience_years,omitempty"`
	ExpectedSalary     *float64 `json:"expected_salary,omitempty"`
	Currency           *string  `json:"currency,omitempty"`
	ResumeFileID       *int64   `json:"resume_file_id,omitempty"`
	CoverLetter        *string  `json:"cover_letter,omitempty"`
	Source             *string  `json:"source,omitempty"`
	ReferrerEmployeeID *int64   `json:"referrer_employee_id,omitempty"`
	Status             *string  `json:"status,omitempty"`
	RejectionReason    *string  `json:"rejection_reason,omitempty"`
	Rating             *int     `json:"rating,omitempty"`
	Notes              *string  `json:"notes,omitempty"`
}

// MoveCandidateRequest represents request to move candidate in pipeline
type MoveCandidateRequest struct {
	Status          string  `json:"status" validate:"required,oneof=new screening interview offer hired rejected withdrawn"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// CandidateFilter represents filter for candidates
type CandidateFilter struct {
	VacancyID          *int64  `json:"vacancy_id,omitempty"`
	Status             *string `json:"status,omitempty"`
	Source             *string `json:"source,omitempty"`
	ReferrerEmployeeID *int64  `json:"referrer_employee_id,omitempty"`
	RatingMin          *int    `json:"rating_min,omitempty"`
	Search             *string `json:"search,omitempty"` // Name, email search
	Limit              int     `json:"limit,omitempty"`
	Offset             int     `json:"offset,omitempty"`
}

// --- Interview DTOs ---

// AddInterviewRequest represents request to schedule interview
type AddInterviewRequest struct {
	CandidateID     int64     `json:"candidate_id" validate:"required"`
	VacancyID       int64     `json:"vacancy_id" validate:"required"`
	InterviewType   string    `json:"interview_type" validate:"required,oneof=phone video onsite technical hr final"`
	ScheduledAt     time.Time `json:"scheduled_at" validate:"required"`
	DurationMinutes int       `json:"duration_minutes" validate:"required,min=15"`
	Location        *string   `json:"location,omitempty"`
	InterviewerIDs  []int64   `json:"interviewer_ids" validate:"required,min=1"`
	Notes           *string   `json:"notes,omitempty"`
}

// EditInterviewRequest represents request to update interview
type EditInterviewRequest struct {
	InterviewType   *string    `json:"interview_type,omitempty"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	Location        *string    `json:"location,omitempty"`
	InterviewerIDs  []int64    `json:"interviewer_ids,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
}

// CompleteInterviewRequest represents request to complete interview
type CompleteInterviewRequest struct {
	OverallRating  int     `json:"overall_rating" validate:"required,min=1,max=5"`
	Feedback       *string `json:"feedback,omitempty"`
	Recommendation string  `json:"recommendation" validate:"required,oneof=hire maybe reject"`
	Notes          *string `json:"notes,omitempty"`
}

// CancelInterviewRequest represents request to cancel interview
type CancelInterviewRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// InterviewFilter represents filter for interviews
type InterviewFilter struct {
	CandidateID   *int64     `json:"candidate_id,omitempty"`
	VacancyID     *int64     `json:"vacancy_id,omitempty"`
	InterviewerID *int64     `json:"interviewer_id,omitempty"`
	InterviewType *string    `json:"interview_type,omitempty"`
	Status        *string    `json:"status,omitempty"`
	FromDate      *time.Time `json:"from_date,omitempty"`
	ToDate        *time.Time `json:"to_date,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	Offset        int        `json:"offset,omitempty"`
}
