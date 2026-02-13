package recruiting

import (
	"encoding/json"
	"time"
)

type Vacancy struct {
	ID                 int64           `json:"id"`
	Title              string          `json:"title"`
	DepartmentID       int64           `json:"department_id"`
	DepartmentName     string          `json:"department_name"`
	PositionID         int64           `json:"position_id"`
	PositionName       string          `json:"position_name"`
	Description        *string         `json:"description,omitempty"`
	Requirements       *string         `json:"requirements,omitempty"`
	SalaryFrom         *float64        `json:"salary_from,omitempty"`
	SalaryTo           *float64        `json:"salary_to,omitempty"`
	EmploymentType     string          `json:"employment_type"`
	ExperienceRequired *string         `json:"experience_required,omitempty"`
	EducationRequired  *string         `json:"education_required,omitempty"`
	Skills             json.RawMessage `json:"skills"`
	Status             string          `json:"status"`
	Priority           string          `json:"priority"`
	PublishedAt        *time.Time      `json:"published_at,omitempty"`
	Deadline           *string         `json:"deadline,omitempty"`
	ResponsibleID      *int64          `json:"responsible_id,omitempty"`
	CreatedBy          *int64          `json:"created_by,omitempty"`
	CandidatesCount    int             `json:"candidates_count"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type CandidateListItem struct {
	ID                int64           `json:"id"`
	VacancyID         int64           `json:"vacancy_id"`
	Name              string          `json:"name"`
	Email             *string         `json:"email,omitempty"`
	Phone             *string         `json:"phone,omitempty"`
	Source            string          `json:"source"`
	Status            string          `json:"status"`
	Stage             string          `json:"stage"`
	ResumeURL         *string         `json:"resume_url,omitempty"`
	PhotoURL          *string         `json:"photo_url,omitempty"`
	Skills            json.RawMessage `json:"skills"`
	Languages         json.RawMessage `json:"languages"`
	SalaryExpectation *float64        `json:"salary_expectation,omitempty"`
	Notes             *string         `json:"notes,omitempty"`
	Rating            *int            `json:"rating,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type Candidate struct {
	CandidateListItem
	Education      []Education  `json:"education"`
	WorkExperience []Experience `json:"work_experience"`
}

type Education struct {
	ID           int64   `json:"id"`
	CandidateID  int64   `json:"candidate_id"`
	Institution  string  `json:"institution"`
	Degree       *string `json:"degree,omitempty"`
	FieldOfStudy *string `json:"field_of_study,omitempty"`
	StartDate    *string `json:"start_date,omitempty"`
	EndDate      *string `json:"end_date,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type Experience struct {
	ID          int64   `json:"id"`
	CandidateID int64   `json:"candidate_id"`
	Company     string  `json:"company"`
	Position    *string `json:"position,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	Description *string `json:"description,omitempty"`
}

type Interview struct {
	ID              int64           `json:"id"`
	CandidateID     int64           `json:"candidate_id"`
	VacancyID       int64           `json:"vacancy_id"`
	Type            string          `json:"type"`
	ScheduledAt     time.Time       `json:"scheduled_at"`
	DurationMinutes int             `json:"duration_minutes"`
	Location        *string         `json:"location,omitempty"`
	Interviewers    json.RawMessage `json:"interviewers"`
	Status          string          `json:"status"`
	OverallRating   *int            `json:"overall_rating,omitempty"`
	Recommendation  *string         `json:"recommendation,omitempty"`
	Feedback        *string         `json:"feedback,omitempty"`
	Scores          json.RawMessage `json:"scores"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
