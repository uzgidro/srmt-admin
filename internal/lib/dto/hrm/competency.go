package hrm

import "time"

// --- Competency Category DTOs ---

// AddCompetencyCategoryRequest represents request to create category
type AddCompetencyCategoryRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
	SortOrder   int     `json:"sort_order"`
}

// EditCompetencyCategoryRequest represents request to update category
type EditCompetencyCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	SortOrder   *int    `json:"sort_order,omitempty"`
}

// --- Competency DTOs ---

// AddCompetencyRequest represents request to create competency
type AddCompetencyRequest struct {
	CategoryID           int     `json:"category_id" validate:"required"`
	Name                 string  `json:"name" validate:"required"`
	Code                 *string `json:"code,omitempty"`
	Description          *string `json:"description,omitempty"`
	BehavioralIndicators *string `json:"behavioral_indicators,omitempty"`
}

// EditCompetencyRequest represents request to update competency
type EditCompetencyRequest struct {
	CategoryID           *int    `json:"category_id,omitempty"`
	Name                 *string `json:"name,omitempty"`
	Code                 *string `json:"code,omitempty"`
	Description          *string `json:"description,omitempty"`
	BehavioralIndicators *string `json:"behavioral_indicators,omitempty"`
	IsActive             *bool   `json:"is_active,omitempty"`
}

// CompetencyFilter represents filter for competencies
type CompetencyFilter struct {
	CategoryID *int    `json:"category_id,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
	Search     *string `json:"search,omitempty"` // Name, code search
}

// --- Competency Level DTOs ---

// AddCompetencyLevelRequest represents request to add level
type AddCompetencyLevelRequest struct {
	CompetencyID int     `json:"competency_id" validate:"required"`
	Level        int     `json:"level" validate:"required,min=1,max=5"`
	Name         string  `json:"name" validate:"required"`
	Description  *string `json:"description,omitempty"`
}

// EditCompetencyLevelRequest represents request to update level
type EditCompetencyLevelRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// BulkCompetencyLevelsRequest represents request to set all levels
type BulkCompetencyLevelsRequest struct {
	CompetencyID int                   `json:"competency_id" validate:"required"`
	Levels       []CompetencyLevelData `json:"levels" validate:"required,min=1,max=5"`
}

// CompetencyLevelData represents single level in bulk request
type CompetencyLevelData struct {
	Level       int     `json:"level" validate:"required,min=1,max=5"`
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
}

// --- Competency Matrix DTOs ---

// AddCompetencyMatrixRequest represents request to add matrix entry
type AddCompetencyMatrixRequest struct {
	PositionID    int64   `json:"position_id" validate:"required"`
	CompetencyID  int     `json:"competency_id" validate:"required"`
	RequiredLevel int     `json:"required_level" validate:"required,min=1,max=5"`
	IsMandatory   bool    `json:"is_mandatory"`
	Weight        float64 `json:"weight" validate:"omitempty,min=0,max=10"`
}

// EditCompetencyMatrixRequest represents request to update matrix entry
type EditCompetencyMatrixRequest struct {
	RequiredLevel *int     `json:"required_level,omitempty"`
	IsMandatory   *bool    `json:"is_mandatory,omitempty"`
	Weight        *float64 `json:"weight,omitempty"`
}

// BulkCompetencyMatrixRequest represents request to set matrix for position
type BulkCompetencyMatrixRequest struct {
	PositionID int64                   `json:"position_id" validate:"required"`
	Entries    []CompetencyMatrixEntry `json:"entries" validate:"required,min=1"`
}

// CompetencyMatrixEntry represents single entry in bulk request
type CompetencyMatrixEntry struct {
	CompetencyID  int     `json:"competency_id" validate:"required"`
	RequiredLevel int     `json:"required_level" validate:"required,min=1,max=5"`
	IsMandatory   bool    `json:"is_mandatory"`
	Weight        float64 `json:"weight"`
}

// CompetencyMatrixFilter represents filter for matrix entries
type CompetencyMatrixFilter struct {
	PositionID   *int64 `json:"position_id,omitempty"`
	CompetencyID *int   `json:"competency_id,omitempty"`
	IsMandatory  *bool  `json:"is_mandatory,omitempty"`
}

// --- Competency Assessment DTOs ---

// AddAssessmentRequest represents request to create assessment
type AddAssessmentRequest struct {
	EmployeeID            int64     `json:"employee_id" validate:"required"`
	AssessmentType        string    `json:"assessment_type" validate:"required,oneof=self manager peer 360"`
	AssessmentPeriodStart time.Time `json:"assessment_period_start" validate:"required"`
	AssessmentPeriodEnd   time.Time `json:"assessment_period_end" validate:"required"`
	AssessorID            *int64    `json:"assessor_id,omitempty"`
}

// EditAssessmentRequest represents request to update assessment
type EditAssessmentRequest struct {
	Status       *string  `json:"status,omitempty"`
	OverallScore *float64 `json:"overall_score,omitempty"`
	Notes        *string  `json:"notes,omitempty"`
}

// StartAssessmentRequest represents request to start assessment
type StartAssessmentRequest struct {
	// Empty - just triggers status change
}

// CompleteAssessmentRequest represents request to complete assessment
type CompleteAssessmentRequest struct {
	OverallScore *float64 `json:"overall_score,omitempty"`
	Notes        *string  `json:"notes,omitempty"`
}

// AssessmentFilter represents filter for assessments
type AssessmentFilter struct {
	EmployeeID     *int64     `json:"employee_id,omitempty"`
	AssessorID     *int64     `json:"assessor_id,omitempty"`
	AssessmentType *string    `json:"assessment_type,omitempty"`
	Status         *string    `json:"status,omitempty"`
	FromDate       *time.Time `json:"from_date,omitempty"`
	ToDate         *time.Time `json:"to_date,omitempty"`
	Limit          int        `json:"limit,omitempty"`
	Offset         int        `json:"offset,omitempty"`
}

// --- Competency Score DTOs ---

// AddScoreRequest represents request to add score
type AddScoreRequest struct {
	AssessmentID int64   `json:"assessment_id" validate:"required"`
	CompetencyID int     `json:"competency_id" validate:"required"`
	Score        int     `json:"score" validate:"required,min=1,max=5"`
	Evidence     *string `json:"evidence,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

// EditScoreRequest represents request to update score
type EditScoreRequest struct {
	Score    *int    `json:"score,omitempty"`
	Evidence *string `json:"evidence,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

// BulkScoresRequest represents request to submit multiple scores
type BulkScoresRequest struct {
	AssessmentID int64       `json:"assessment_id" validate:"required"`
	Scores       []ScoreData `json:"scores" validate:"required,min=1"`
}

// ScoreData represents single score in bulk request
type ScoreData struct {
	CompetencyID int     `json:"competency_id" validate:"required"`
	Score        int     `json:"score" validate:"required,min=1,max=5"`
	Evidence     *string `json:"evidence,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

// ScoreFilter represents filter for scores
type ScoreFilter struct {
	AssessmentID *int64 `json:"assessment_id,omitempty"`
	CompetencyID *int   `json:"competency_id,omitempty"`
}
