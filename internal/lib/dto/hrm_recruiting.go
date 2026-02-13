package dto

import "encoding/json"

// --- Vacancies ---

type CreateVacancyRequest struct {
	Title              string          `json:"title" validate:"required"`
	DepartmentID       int64           `json:"department_id" validate:"required"`
	PositionID         int64           `json:"position_id" validate:"required"`
	Description        *string         `json:"description,omitempty"`
	Requirements       *string         `json:"requirements,omitempty"`
	SalaryFrom         *float64        `json:"salary_from,omitempty"`
	SalaryTo           *float64        `json:"salary_to,omitempty"`
	EmploymentType     string          `json:"employment_type" validate:"required,oneof=full_time part_time contract internship"`
	ExperienceRequired *string         `json:"experience_required,omitempty"`
	EducationRequired  *string         `json:"education_required,omitempty"`
	Skills             json.RawMessage `json:"skills,omitempty"`
	Priority           string          `json:"priority" validate:"required,oneof=low medium high urgent"`
	Deadline           *string         `json:"deadline,omitempty"`
	ResponsibleID      *int64          `json:"responsible_id,omitempty"`
}

type UpdateVacancyRequest struct {
	Title              *string          `json:"title,omitempty"`
	DepartmentID       *int64           `json:"department_id,omitempty"`
	PositionID         *int64           `json:"position_id,omitempty"`
	Description        *string          `json:"description,omitempty"`
	Requirements       *string          `json:"requirements,omitempty"`
	SalaryFrom         *float64         `json:"salary_from,omitempty"`
	SalaryTo           *float64         `json:"salary_to,omitempty"`
	EmploymentType     *string          `json:"employment_type,omitempty" validate:"omitempty,oneof=full_time part_time contract internship"`
	ExperienceRequired *string          `json:"experience_required,omitempty"`
	EducationRequired  *string          `json:"education_required,omitempty"`
	Skills             *json.RawMessage `json:"skills,omitempty"`
	Priority           *string          `json:"priority,omitempty" validate:"omitempty,oneof=low medium high urgent"`
	Deadline           *string          `json:"deadline,omitempty"`
	ResponsibleID      *int64           `json:"responsible_id,omitempty"`
}

type VacancyFilters struct {
	DepartmentID   *int64
	Status         *string
	Priority       *string
	EmploymentType *string
	Search         *string
}

// --- Candidates ---

type CreateCandidateRequest struct {
	VacancyID         int64             `json:"vacancy_id" validate:"required"`
	Name              string            `json:"name" validate:"required"`
	Email             *string           `json:"email,omitempty"`
	Phone             *string           `json:"phone,omitempty"`
	Source            string            `json:"source" validate:"required,oneof=website linkedin referral agency job_board social_media university internal other"`
	ResumeURL         *string           `json:"resume_url,omitempty"`
	PhotoURL          *string           `json:"photo_url,omitempty"`
	Skills            json.RawMessage   `json:"skills,omitempty"`
	Languages         json.RawMessage   `json:"languages,omitempty"`
	SalaryExpectation *float64          `json:"salary_expectation,omitempty"`
	Notes             *string           `json:"notes,omitempty"`
	Education         []EducationInput  `json:"education,omitempty"`
	WorkExperience    []ExperienceInput `json:"work_experience,omitempty"`
}

type UpdateCandidateRequest struct {
	Name              *string            `json:"name,omitempty"`
	Email             *string            `json:"email,omitempty"`
	Phone             *string            `json:"phone,omitempty"`
	Source            *string            `json:"source,omitempty" validate:"omitempty,oneof=website linkedin referral agency job_board social_media university internal other"`
	ResumeURL         *string            `json:"resume_url,omitempty"`
	PhotoURL          *string            `json:"photo_url,omitempty"`
	Skills            *json.RawMessage   `json:"skills,omitempty"`
	Languages         *json.RawMessage   `json:"languages,omitempty"`
	SalaryExpectation *float64           `json:"salary_expectation,omitempty"`
	Notes             *string            `json:"notes,omitempty"`
	Rating            *int               `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	Education         *[]EducationInput  `json:"education,omitempty"`
	WorkExperience    *[]ExperienceInput `json:"work_experience,omitempty"`
}

type ChangeCandidateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=new screening phone_interview assessment interview offer hired rejected withdrawn blacklisted"`
	Stage  string `json:"stage" validate:"required,oneof=sourcing screening interview assessment offer hiring onboarding"`
}

type CandidateFilters struct {
	VacancyID *int64
	Status    *string
	Stage     *string
	Source    *string
	Search    *string
}

type EducationInput struct {
	Institution  string  `json:"institution" validate:"required"`
	Degree       *string `json:"degree,omitempty"`
	FieldOfStudy *string `json:"field_of_study,omitempty"`
	StartDate    *string `json:"start_date,omitempty"`
	EndDate      *string `json:"end_date,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type ExperienceInput struct {
	Company     string  `json:"company" validate:"required"`
	Position    *string `json:"position,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	Description *string `json:"description,omitempty"`
}

// --- Interviews ---

type CreateInterviewRequest struct {
	CandidateID     int64           `json:"candidate_id" validate:"required"`
	VacancyID       int64           `json:"vacancy_id" validate:"required"`
	Type            string          `json:"type" validate:"required,oneof=phone video in_person technical hr final group"`
	ScheduledAt     string          `json:"scheduled_at" validate:"required"`
	DurationMinutes int             `json:"duration_minutes" validate:"required,min=1"`
	Location        *string         `json:"location,omitempty"`
	Interviewers    json.RawMessage `json:"interviewers,omitempty"`
}

type UpdateInterviewRequest struct {
	Type            *string          `json:"type,omitempty" validate:"omitempty,oneof=phone video in_person technical hr final group"`
	ScheduledAt     *string          `json:"scheduled_at,omitempty"`
	DurationMinutes *int             `json:"duration_minutes,omitempty" validate:"omitempty,min=1"`
	Location        *string          `json:"location,omitempty"`
	Interviewers    *json.RawMessage `json:"interviewers,omitempty"`
	Status          *string          `json:"status,omitempty" validate:"omitempty,oneof=scheduled in_progress completed cancelled no_show rescheduled"`
	OverallRating   *int             `json:"overall_rating,omitempty" validate:"omitempty,min=1,max=5"`
	Recommendation  *string          `json:"recommendation,omitempty" validate:"omitempty,oneof=strong_hire hire no_hire strong_no_hire"`
	Feedback        *string          `json:"feedback,omitempty"`
	Scores          *json.RawMessage `json:"scores,omitempty"`
}

type InterviewFilters struct {
	CandidateID *int64
	VacancyID   *int64
	Status      *string
	Type        *string
}
