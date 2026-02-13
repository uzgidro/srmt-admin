package dto

import "encoding/json"

// --- Competencies ---

type CreateCompetencyRequest struct {
	Name        string          `json:"name" validate:"required"`
	Description *string         `json:"description,omitempty"`
	Category    string          `json:"category" validate:"required,oneof=professional personal managerial technical communication leadership"`
	Levels      json.RawMessage `json:"levels" validate:"required"`
	PositionIDs []int64         `json:"position_ids,omitempty"`
}

type UpdateCompetencyRequest struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Category    *string          `json:"category,omitempty" validate:"omitempty,oneof=professional personal managerial technical communication leadership"`
	Levels      *json.RawMessage `json:"levels,omitempty"`
}

type CompetencyFilters struct {
	Category *string
	Search   *string
}

// --- Assessments ---

type CreateAssessmentRequest struct {
	Name         string                      `json:"name" validate:"required"`
	Description  *string                     `json:"description,omitempty"`
	StartDate    string                      `json:"start_date" validate:"required"`
	EndDate      string                      `json:"end_date" validate:"required"`
	Competencies []AssessmentCompetencyInput `json:"competencies" validate:"required,min=1,dive"`
	Candidates   []int64                     `json:"candidates" validate:"required,min=1"`
	Assessors    []AssessmentAssessorInput   `json:"assessors" validate:"required,min=1,dive"`
}

type AssessmentCompetencyInput struct {
	CompetencyID  int64   `json:"competency_id" validate:"required"`
	Weight        float64 `json:"weight" validate:"required,gt=0"`
	RequiredLevel int     `json:"required_level" validate:"required,min=1,max=5"`
}

type AssessmentAssessorInput struct {
	EmployeeID int64  `json:"employee_id" validate:"required"`
	Role       string `json:"role" validate:"required,oneof=manager peer self expert subordinate"`
}

type UpdateAssessmentRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

type AssessmentFilters struct {
	Status *string
	Search *string
}

type SubmitScoresRequest struct {
	Scores []ScoreInput `json:"scores" validate:"required,min=1,dive"`
}

type ScoreInput struct {
	CandidateID  int64   `json:"candidate_id" validate:"required"`
	CompetencyID int64   `json:"competency_id" validate:"required"`
	Score        int     `json:"score" validate:"required,min=1,max=5"`
	Comment      *string `json:"comment,omitempty"`
}
