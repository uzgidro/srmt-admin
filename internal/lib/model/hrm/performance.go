package hrm

import "time"

// PerformanceReview represents a performance review cycle
type PerformanceReview struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	// Review period
	ReviewType        string    `json:"review_type"`
	ReviewPeriodStart time.Time `json:"review_period_start"`
	ReviewPeriodEnd   time.Time `json:"review_period_end"`

	// Status and workflow
	Status                string     `json:"status"`
	SelfReviewDeadline    *time.Time `json:"self_review_deadline,omitempty"`
	ManagerReviewDeadline *time.Time `json:"manager_review_deadline,omitempty"`

	// Self review
	SelfReviewStartedAt   *time.Time `json:"self_review_started_at,omitempty"`
	SelfReviewCompletedAt *time.Time `json:"self_review_completed_at,omitempty"`
	SelfAssessment        *string    `json:"self_assessment,omitempty"`
	SelfRating            *int       `json:"self_rating,omitempty"`

	// Manager review
	ReviewerID               *int64     `json:"reviewer_id,omitempty"`
	ManagerReviewStartedAt   *time.Time `json:"manager_review_started_at,omitempty"`
	ManagerReviewCompletedAt *time.Time `json:"manager_review_completed_at,omitempty"`
	ManagerAssessment        *string    `json:"manager_assessment,omitempty"`
	ManagerRating            *int       `json:"manager_rating,omitempty"`

	// Final rating
	FinalRating      *int       `json:"final_rating,omitempty"`
	FinalRatingLabel *string    `json:"final_rating_label,omitempty"`
	CalibratedBy     *int64     `json:"calibrated_by,omitempty"`
	CalibratedAt     *time.Time `json:"calibrated_at,omitempty"`

	// Summary
	Achievements               *string `json:"achievements,omitempty"`
	AreasForImprovement        *string `json:"areas_for_improvement,omitempty"`
	DevelopmentRecommendations *string `json:"development_recommendations,omitempty"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Notes       *string    `json:"notes,omitempty"`

	// Enriched
	Employee *Employee         `json:"employee,omitempty"`
	Reviewer *Employee         `json:"reviewer,omitempty"`
	Goals    []PerformanceGoal `json:"goals,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// ReviewType constants
const (
	ReviewTypeAnnual    = "annual"
	ReviewTypeMidYear   = "mid_year"
	ReviewTypeQuarterly = "quarterly"
	ReviewTypeProbation = "probation"
)

// ReviewStatus constants
const (
	ReviewStatusPending       = "pending"
	ReviewStatusSelfReview    = "self_review"
	ReviewStatusManagerReview = "manager_review"
	ReviewStatusCalibration   = "calibration"
	ReviewStatusCompleted     = "completed"
)

// RatingLabel constants
const (
	RatingLabelExceptional      = "Exceptional"
	RatingLabelExceedsExpect    = "Exceeds Expectations"
	RatingLabelMeetsExpect      = "Meets Expectations"
	RatingLabelNeedsImprovement = "Needs Improvement"
	RatingLabelUnsatisfactory   = "Unsatisfactory"
)

// GetRatingLabel returns label for numeric rating
func GetRatingLabel(rating int) string {
	switch rating {
	case 5:
		return RatingLabelExceptional
	case 4:
		return RatingLabelExceedsExpect
	case 3:
		return RatingLabelMeetsExpect
	case 2:
		return RatingLabelNeedsImprovement
	case 1:
		return RatingLabelUnsatisfactory
	default:
		return ""
	}
}

// PerformanceGoal represents performance goals/objectives
type PerformanceGoal struct {
	ID         int64  `json:"id"`
	EmployeeID int64  `json:"employee_id"`
	ReviewID   *int64 `json:"review_id,omitempty"`

	// Goal details
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`

	// SMART criteria
	SuccessCriteria *string `json:"success_criteria,omitempty"`
	Metrics         *string `json:"metrics,omitempty"`

	// Alignment
	AlignedTo *string `json:"aligned_to,omitempty"`
	Weight    float64 `json:"weight"`

	// Timeline
	StartDate  *time.Time `json:"start_date,omitempty"`
	TargetDate *time.Time `json:"target_date,omitempty"`

	// Progress
	Status      string     `json:"status"`
	Progress    int        `json:"progress"` // 0-100
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Self assessment
	SelfRating   *int    `json:"self_rating,omitempty"`
	SelfComments *string `json:"self_comments,omitempty"`

	// Manager assessment
	ManagerRating   *int    `json:"manager_rating,omitempty"`
	ManagerComments *string `json:"manager_comments,omitempty"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee *Employee          `json:"employee,omitempty"`
	Review   *PerformanceReview `json:"review,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// GoalCategory constants
const (
	GoalCategoryBusiness    = "business"
	GoalCategoryDevelopment = "development"
	GoalCategoryTeam        = "team"
	GoalCategoryProcess     = "process"
)

// GoalStatus constants
const (
	GoalStatusNotStarted = "not_started"
	GoalStatusInProgress = "in_progress"
	GoalStatusCompleted  = "completed"
	GoalStatusCancelled  = "cancelled"
	GoalStatusDeferred   = "deferred"
)

// KPI represents Key Performance Indicator
type KPI struct {
	ID         int64 `json:"id"`
	EmployeeID int64 `json:"employee_id"`

	// Definition
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`

	// Measurement
	MeasurementUnit *string  `json:"measurement_unit,omitempty"`
	TargetValue     float64  `json:"target_value"`
	MinThreshold    *float64 `json:"min_threshold,omitempty"`
	MaxThreshold    *float64 `json:"max_threshold,omitempty"`

	// Period
	Year    int  `json:"year"`
	Month   *int `json:"month,omitempty"`
	Quarter *int `json:"quarter,omitempty"`

	// Results
	ActualValue        *float64 `json:"actual_value,omitempty"`
	AchievementPercent *float64 `json:"achievement_percent,omitempty"`

	// Rating
	Rating *int    `json:"rating,omitempty"`
	Weight float64 `json:"weight"`

	Notes *string `json:"notes,omitempty"`

	// Enriched
	Employee *Employee `json:"employee,omitempty"`

	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// KPICategory constants
const (
	KPICategorySales        = "sales"
	KPICategoryQuality      = "quality"
	KPICategoryProductivity = "productivity"
	KPICategoryCustomer     = "customer"
	KPICategoryFinancial    = "financial"
)

// MeasurementUnit constants
const (
	MeasurementUnitPercent  = "percent"
	MeasurementUnitCount    = "count"
	MeasurementUnitCurrency = "currency"
	MeasurementUnitScore    = "score"
	MeasurementUnitDays     = "days"
	MeasurementUnitHours    = "hours"
)

// CalculateAchievement calculates achievement percentage
func (k *KPI) CalculateAchievement() *float64 {
	if k.ActualValue == nil || k.TargetValue == 0 {
		return nil
	}
	achievement := (*k.ActualValue / k.TargetValue) * 100
	return &achievement
}

// PerformanceStats represents performance metrics
type PerformanceStats struct {
	TotalReviews          int         `json:"total_reviews"`
	PendingReviews        int         `json:"pending_reviews"`
	CompletedReviews      int         `json:"completed_reviews"`
	AverageRating         float64     `json:"average_rating"`
	DistributionByRating  map[int]int `json:"distribution_by_rating"`
	GoalsCompletionRate   float64     `json:"goals_completion_rate"`
	AverageKPIAchievement float64     `json:"average_kpi_achievement"`
}
