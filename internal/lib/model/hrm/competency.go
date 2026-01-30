package hrm

import (
	"time"

	"srmt-admin/internal/lib/model/position"
)

// CompetencyCategory represents category of competencies
type CompetencyCategory struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	SortOrder   int     `json:"sort_order"`

	// Enriched
	Competencies []Competency `json:"competencies,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// Competency represents a competency definition
type Competency struct {
	ID         int     `json:"id"`
	CategoryID int     `json:"category_id"`
	Name       string  `json:"name"`
	Code       *string `json:"code,omitempty"`

	Description          *string `json:"description,omitempty"`
	BehavioralIndicators *string `json:"behavioral_indicators,omitempty"`

	IsActive bool `json:"is_active"`

	// Enriched
	Category *CompetencyCategory `json:"category,omitempty"`
	Levels   []CompetencyLevel   `json:"levels,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// CompetencyLevel represents proficiency level for a competency
type CompetencyLevel struct {
	ID           int `json:"id"`
	CompetencyID int `json:"competency_id"`

	Level       int     `json:"level"` // 1-5
	Name        string  `json:"name"`  // Beginner, Developing, Proficient, Advanced, Expert
	Description *string `json:"description,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// Default level names
const (
	CompetencyLevelBeginner   = "Beginner"
	CompetencyLevelDeveloping = "Developing"
	CompetencyLevelProficient = "Proficient"
	CompetencyLevelAdvanced   = "Advanced"
	CompetencyLevelExpert     = "Expert"
)

// CompetencyMatrix represents required competency level per position
type CompetencyMatrix struct {
	ID           int64 `json:"id"`
	PositionID   int64 `json:"position_id"`
	CompetencyID int   `json:"competency_id"`

	RequiredLevel int     `json:"required_level"` // 1-5
	IsMandatory   bool    `json:"is_mandatory"`
	Weight        float64 `json:"weight"`

	// Enriched
	Position   *position.Model `json:"position,omitempty"`
	Competency *Competency     `json:"competency,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// CompetencyAssessment represents an assessment session
type CompetencyAssessment struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	AssessmentType        string    `json:"assessment_type"`
	AssessmentPeriodStart time.Time `json:"assessment_period_start"`
	AssessmentPeriodEnd   time.Time `json:"assessment_period_end"`

	// Status
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Assessor
	AssessorID *int64 `json:"assessor_id,omitempty"`

	// Results
	OverallScore *float64 `json:"overall_score,omitempty"`
	Notes        *string  `json:"notes,omitempty"`

	// Enriched
	Employee *Employee         `json:"employee,omitempty"`
	Assessor *Employee         `json:"assessor,omitempty"`
	Scores   []CompetencyScore `json:"scores,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// AssessmentType constants
const (
	AssessmentTypeSelf    = "self"
	AssessmentTypeManager = "manager"
	AssessmentTypePeer    = "peer"
	AssessmentType360     = "360"
)

// AssessmentStatus constants
const (
	AssessmentStatusPending    = "pending"
	AssessmentStatusInProgress = "in_progress"
	AssessmentStatusCompleted  = "completed"
)

// CompetencyScore represents individual score in assessment
type CompetencyScore struct {
	ID           int64 `json:"id"`
	AssessmentID int64 `json:"assessment_id"`
	CompetencyID int   `json:"competency_id"`

	Score    int     `json:"score"` // 1-5
	Evidence *string `json:"evidence,omitempty"`
	Notes    *string `json:"notes,omitempty"`

	// Enriched
	Competency *Competency `json:"competency,omitempty"`

	// Calculated (from matrix)
	RequiredLevel *int `json:"required_level,omitempty"`
	Gap           *int `json:"gap,omitempty"` // score - required_level

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// EmployeeCompetencyProfile represents aggregated competency data for employee
type EmployeeCompetencyProfile struct {
	EmployeeID int64           `json:"employee_id"`
	Employee   *Employee       `json:"employee,omitempty"`
	Position   *position.Model `json:"position,omitempty"`

	// Latest assessment
	LatestAssessment *CompetencyAssessment `json:"latest_assessment,omitempty"`

	// Required vs Actual
	CompetencyGaps []CompetencyGap `json:"competency_gaps,omitempty"`

	// Summary
	OverallScore    *float64 `json:"overall_score,omitempty"`
	OverallRequired *float64 `json:"overall_required,omitempty"`
	GapPercentage   *float64 `json:"gap_percentage,omitempty"`
}

// CompetencyGap represents gap between required and actual level
type CompetencyGap struct {
	CompetencyID  int         `json:"competency_id"`
	Competency    *Competency `json:"competency,omitempty"`
	RequiredLevel int         `json:"required_level"`
	ActualLevel   *int        `json:"actual_level,omitempty"`
	Gap           int         `json:"gap"`
	IsMandatory   bool        `json:"is_mandatory"`
}

// CompetencyStats represents competency metrics
type CompetencyStats struct {
	TotalCompetencies  int     `json:"total_competencies"`
	TotalAssessments   int     `json:"total_assessments"`
	PendingAssessments int     `json:"pending_assessments"`
	AverageScore       float64 `json:"average_score"`
	EmployeesWithGaps  int     `json:"employees_with_gaps"`
	CriticalGapsCount  int     `json:"critical_gaps_count"` // Mandatory competencies with gap > 1
}
