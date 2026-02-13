package competency

import (
	"encoding/json"
	"time"
)

type CompetencyLevel struct {
	Level       int      `json:"level"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Indicators  []string `json:"indicators"`
}

type Competency struct {
	ID                   int64           `json:"id"`
	Name                 string          `json:"name"`
	Description          *string         `json:"description,omitempty"`
	Category             string          `json:"category"`
	Levels               json.RawMessage `json:"levels"`
	RequiredForPositions json.RawMessage `json:"required_for_positions"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type AssessmentSession struct {
	ID           int64                   `json:"id"`
	Name         string                  `json:"name"`
	Description  *string                 `json:"description,omitempty"`
	Status       string                  `json:"status"`
	StartDate    string                  `json:"start_date"`
	EndDate      string                  `json:"end_date"`
	CreatedBy    *int64                  `json:"created_by,omitempty"`
	Competencies []*AssessmentCompetency `json:"competencies"`
	Candidates   []*AssessmentCandidate  `json:"candidates"`
	Assessors    []*AssessmentAssessor   `json:"assessors"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

type AssessmentCompetency struct {
	ID             int64   `json:"id"`
	SessionID      int64   `json:"session_id"`
	CompetencyID   int64   `json:"competency_id"`
	CompetencyName string  `json:"competency_name"`
	Weight         float64 `json:"weight"`
	RequiredLevel  int     `json:"required_level"`
}

type AssessmentCandidate struct {
	ID         int64  `json:"id"`
	SessionID  int64  `json:"session_id"`
	EmployeeID int64  `json:"employee_id"`
	Name       string `json:"name"`
	Position   string `json:"position"`
	Department string `json:"department"`
	Status     string `json:"status"`
}

type AssessmentAssessor struct {
	ID         int64  `json:"id"`
	SessionID  int64  `json:"session_id"`
	EmployeeID int64  `json:"employee_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
}

type AssessmentScore struct {
	ID           int64     `json:"id"`
	SessionID    int64     `json:"session_id"`
	CandidateID  int64     `json:"candidate_id"`
	AssessorID   int64     `json:"assessor_id"`
	CompetencyID int64     `json:"competency_id"`
	Score        int       `json:"score"`
	Comment      *string   `json:"comment,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type GapAnalysis struct {
	EmployeeID   int64      `json:"employee_id"`
	EmployeeName string     `json:"employee_name"`
	Position     string     `json:"position"`
	Items        []*GapItem `json:"items"`
}

type GapItem struct {
	CompetencyID   int64   `json:"competency_id"`
	CompetencyName string  `json:"competency_name"`
	Category       string  `json:"category"`
	RequiredLevel  int     `json:"required_level"`
	CurrentLevel   float64 `json:"current_level"`
	Gap            float64 `json:"gap"`
}

type CompetencyMatrix struct {
	PositionID   int64                   `json:"position_id"`
	PositionName string                  `json:"position_name"`
	Items        []*CompetencyMatrixItem `json:"items"`
}

type CompetencyMatrixItem struct {
	CompetencyID   int64  `json:"competency_id"`
	CompetencyName string `json:"competency_name"`
	Category       string `json:"category"`
	RequiredLevel  int    `json:"required_level"`
}

type CompetencyReport struct {
	TotalAssessments int              `json:"total_assessments"`
	CompletedCount   int              `json:"completed_count"`
	AverageScore     float64          `json:"average_score"`
	ByCategory       []*CategoryScore `json:"by_category"`
	TopGaps          []*GapItem       `json:"top_gaps"`
}

type CategoryScore struct {
	Category      string  `json:"category"`
	AverageScore  float64 `json:"average_score"`
	EmployeeCount int     `json:"employee_count"`
}
