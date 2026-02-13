package training

import (
	"encoding/json"
	"time"
)

type Training struct {
	ID                  int64           `json:"id"`
	Title               string          `json:"title"`
	Description         *string         `json:"description,omitempty"`
	Type                string          `json:"type"`
	Status              string          `json:"status"`
	Provider            *string         `json:"provider,omitempty"`
	Trainer             *string         `json:"trainer,omitempty"`
	StartDate           string          `json:"start_date"`
	EndDate             string          `json:"end_date"`
	Location            *string         `json:"location,omitempty"`
	MaxParticipants     int             `json:"max_participants"`
	CurrentParticipants int             `json:"current_participants"`
	Cost                *float64        `json:"cost,omitempty"`
	Mandatory           bool            `json:"mandatory"`
	DepartmentIDs       json.RawMessage `json:"department_ids"`
	CreatedBy           *int64          `json:"created_by,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type Participant struct {
	ID            int64      `json:"id"`
	TrainingID    int64      `json:"training_id"`
	EmployeeID    int64      `json:"employee_id"`
	EmployeeName  string     `json:"employee_name"`
	Status        string     `json:"status"`
	EnrolledAt    time.Time  `json:"enrolled_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Score         *int       `json:"score,omitempty"`
	CertificateID *int64     `json:"certificate_id,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
}

type Certificate struct {
	ID             int64     `json:"id"`
	EmployeeID     int64     `json:"employee_id"`
	EmployeeName   string    `json:"employee_name"`
	TrainingID     *int64    `json:"training_id,omitempty"`
	TrainingTitle  *string   `json:"training_title,omitempty"`
	Title          string    `json:"title"`
	Issuer         *string   `json:"issuer,omitempty"`
	IssueDate      string    `json:"issue_date"`
	ExpiryDate     *string   `json:"expiry_date,omitempty"`
	CertificateURL *string   `json:"certificate_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type DevelopmentPlan struct {
	ID           int64     `json:"id"`
	EmployeeID   int64     `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	Status       string    `json:"status"`
	StartDate    *string   `json:"start_date,omitempty"`
	EndDate      *string   `json:"end_date,omitempty"`
	CreatedBy    *int64    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DevelopmentGoal struct {
	ID          int64      `json:"id"`
	PlanID      int64      `json:"plan_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	TargetDate  *string    `json:"target_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
