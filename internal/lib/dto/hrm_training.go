package dto

import "encoding/json"

// --- Trainings ---

type CreateTrainingRequest struct {
	Title           string          `json:"title" validate:"required"`
	Description     *string         `json:"description,omitempty"`
	Type            string          `json:"type" validate:"required,oneof=internal external online workshop conference certification mentoring"`
	Provider        *string         `json:"provider,omitempty"`
	Trainer         *string         `json:"trainer,omitempty"`
	StartDate       string          `json:"start_date" validate:"required"`
	EndDate         string          `json:"end_date" validate:"required"`
	Location        *string         `json:"location,omitempty"`
	MaxParticipants *int            `json:"max_participants,omitempty"`
	Cost            *float64        `json:"cost,omitempty"`
	Mandatory       *bool           `json:"mandatory,omitempty"`
	DepartmentIDs   json.RawMessage `json:"department_ids,omitempty"`
}

type UpdateTrainingRequest struct {
	Title           *string          `json:"title,omitempty"`
	Description     *string          `json:"description,omitempty"`
	Type            *string          `json:"type,omitempty" validate:"omitempty,oneof=internal external online workshop conference certification mentoring"`
	Status          *string          `json:"status,omitempty" validate:"omitempty,oneof=planned registration_open in_progress completed cancelled"`
	Provider        *string          `json:"provider,omitempty"`
	Trainer         *string          `json:"trainer,omitempty"`
	StartDate       *string          `json:"start_date,omitempty"`
	EndDate         *string          `json:"end_date,omitempty"`
	Location        *string          `json:"location,omitempty"`
	MaxParticipants *int             `json:"max_participants,omitempty"`
	Cost            *float64         `json:"cost,omitempty"`
	Mandatory       *bool            `json:"mandatory,omitempty"`
	DepartmentIDs   *json.RawMessage `json:"department_ids,omitempty"`
}

type TrainingFilters struct {
	Status *string
	Type   *string
	Search *string
}

// --- Participants ---

type AddParticipantRequest struct {
	EmployeeID int64 `json:"employee_id" validate:"required"`
}

type CompleteParticipantRequest struct {
	Score *int    `json:"score,omitempty" validate:"omitempty,min=0,max=100"`
	Notes *string `json:"notes,omitempty"`
}

// --- Development Plans ---

type CreateDevelopmentPlanRequest struct {
	EmployeeID  int64   `json:"employee_id" validate:"required"`
	Title       string  `json:"title" validate:"required"`
	Description *string `json:"description,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

type AddDevelopmentGoalRequest struct {
	Title       string  `json:"title" validate:"required"`
	Description *string `json:"description,omitempty"`
	TargetDate  *string `json:"target_date,omitempty"`
}
