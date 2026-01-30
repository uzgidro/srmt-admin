package hrm

import "time"

// --- Performance Review DTOs ---

// AddPerformanceReviewRequest represents request to create review
type AddPerformanceReviewRequest struct {
	EmployeeID            int64      `json:"employee_id" validate:"required"`
	ReviewType            string     `json:"review_type" validate:"required,oneof=annual mid_year quarterly probation"`
	ReviewPeriodStart     time.Time  `json:"review_period_start" validate:"required"`
	ReviewPeriodEnd       time.Time  `json:"review_period_end" validate:"required"`
	ReviewerID            *int64     `json:"reviewer_id,omitempty"`
	SelfReviewDeadline    *time.Time `json:"self_review_deadline,omitempty"`
	ManagerReviewDeadline *time.Time `json:"manager_review_deadline,omitempty"`
}

// EditPerformanceReviewRequest represents request to update review
type EditPerformanceReviewRequest struct {
	ReviewType            *string    `json:"review_type,omitempty"`
	ReviewPeriodStart     *time.Time `json:"review_period_start,omitempty"`
	ReviewPeriodEnd       *time.Time `json:"review_period_end,omitempty"`
	ReviewerID            *int64     `json:"reviewer_id,omitempty"`
	SelfReviewDeadline    *time.Time `json:"self_review_deadline,omitempty"`
	ManagerReviewDeadline *time.Time `json:"manager_review_deadline,omitempty"`
	Notes                 *string    `json:"notes,omitempty"`
}

// SubmitSelfReviewRequest represents self-review submission
type SubmitSelfReviewRequest struct {
	SelfAssessment string `json:"self_assessment" validate:"required"`
	SelfRating     int    `json:"self_rating" validate:"required,min=1,max=5"`
}

// SubmitManagerReviewRequest represents manager review submission
type SubmitManagerReviewRequest struct {
	ManagerAssessment          string  `json:"manager_assessment" validate:"required"`
	ManagerRating              int     `json:"manager_rating" validate:"required,min=1,max=5"`
	Achievements               *string `json:"achievements,omitempty"`
	AreasForImprovement        *string `json:"areas_for_improvement,omitempty"`
	DevelopmentRecommendations *string `json:"development_recommendations,omitempty"`
}

// CalibrateReviewRequest represents calibration submission
type CalibrateReviewRequest struct {
	FinalRating      int     `json:"final_rating" validate:"required,min=1,max=5"`
	FinalRatingLabel *string `json:"final_rating_label,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

// PerformanceReviewFilter represents filter for reviews
type PerformanceReviewFilter struct {
	EmployeeID     *int64     `json:"employee_id,omitempty"`
	ReviewerID     *int64     `json:"reviewer_id,omitempty"`
	ReviewType     *string    `json:"review_type,omitempty"`
	Status         *string    `json:"status,omitempty"`
	FromDate       *time.Time `json:"from_date,omitempty"`
	ToDate         *time.Time `json:"to_date,omitempty"`
	DepartmentID   *int64     `json:"department_id,omitempty"`
	OrganizationID *int64     `json:"organization_id,omitempty"`
	Limit          int        `json:"limit,omitempty"`
	Offset         int        `json:"offset,omitempty"`
}

// --- Performance Goal DTOs ---

// AddPerformanceGoalRequest represents request to create goal
type AddPerformanceGoalRequest struct {
	EmployeeID      int64      `json:"employee_id" validate:"required"`
	ReviewID        *int64     `json:"review_id,omitempty"`
	Title           string     `json:"title" validate:"required"`
	Description     *string    `json:"description,omitempty"`
	Category        *string    `json:"category,omitempty"`
	SuccessCriteria *string    `json:"success_criteria,omitempty"`
	Metrics         *string    `json:"metrics,omitempty"`
	AlignedTo       *string    `json:"aligned_to,omitempty"`
	Weight          float64    `json:"weight" validate:"omitempty,min=0,max=10"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	TargetDate      *time.Time `json:"target_date,omitempty"`
}

// EditPerformanceGoalRequest represents request to update goal
type EditPerformanceGoalRequest struct {
	Title           *string    `json:"title,omitempty"`
	Description     *string    `json:"description,omitempty"`
	Category        *string    `json:"category,omitempty"`
	SuccessCriteria *string    `json:"success_criteria,omitempty"`
	Metrics         *string    `json:"metrics,omitempty"`
	AlignedTo       *string    `json:"aligned_to,omitempty"`
	Weight          *float64   `json:"weight,omitempty"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	TargetDate      *time.Time `json:"target_date,omitempty"`
	Status          *string    `json:"status,omitempty"`
	Progress        *int       `json:"progress,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
}

// UpdateGoalProgressRequest represents request to update goal progress
type UpdateGoalProgressRequest struct {
	Progress int     `json:"progress" validate:"required,min=0,max=100"`
	Notes    *string `json:"notes,omitempty"`
}

// RateGoalRequest represents request to rate goal
type RateGoalRequest struct {
	SelfRating      *int    `json:"self_rating,omitempty" validate:"omitempty,min=1,max=5"`
	SelfComments    *string `json:"self_comments,omitempty"`
	ManagerRating   *int    `json:"manager_rating,omitempty" validate:"omitempty,min=1,max=5"`
	ManagerComments *string `json:"manager_comments,omitempty"`
}

// PerformanceGoalFilter represents filter for goals
type PerformanceGoalFilter struct {
	EmployeeID *int64     `json:"employee_id,omitempty"`
	ReviewID   *int64     `json:"review_id,omitempty"`
	Status     *string    `json:"status,omitempty"`
	Category   *string    `json:"category,omitempty"`
	FromDate   *time.Time `json:"from_date,omitempty"`
	ToDate     *time.Time `json:"to_date,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// --- KPI DTOs ---

// AddKPIRequest represents request to create KPI
type AddKPIRequest struct {
	EmployeeID      int64    `json:"employee_id" validate:"required"`
	Name            string   `json:"name" validate:"required"`
	Description     *string  `json:"description,omitempty"`
	Category        *string  `json:"category,omitempty"`
	MeasurementUnit *string  `json:"measurement_unit,omitempty"`
	TargetValue     float64  `json:"target_value" validate:"required"`
	MinThreshold    *float64 `json:"min_threshold,omitempty"`
	MaxThreshold    *float64 `json:"max_threshold,omitempty"`
	Year            int      `json:"year" validate:"required"`
	Month           *int     `json:"month,omitempty" validate:"omitempty,min=1,max=12"`
	Quarter         *int     `json:"quarter,omitempty" validate:"omitempty,min=1,max=4"`
	Weight          float64  `json:"weight" validate:"omitempty,min=0,max=10"`
}

// EditKPIRequest represents request to update KPI
type EditKPIRequest struct {
	Name            *string  `json:"name,omitempty"`
	Description     *string  `json:"description,omitempty"`
	Category        *string  `json:"category,omitempty"`
	MeasurementUnit *string  `json:"measurement_unit,omitempty"`
	TargetValue     *float64 `json:"target_value,omitempty"`
	MinThreshold    *float64 `json:"min_threshold,omitempty"`
	MaxThreshold    *float64 `json:"max_threshold,omitempty"`
	Weight          *float64 `json:"weight,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
}

// UpdateKPIValueRequest represents request to update KPI actual value
type UpdateKPIValueRequest struct {
	ActualValue float64 `json:"actual_value" validate:"required"`
	Notes       *string `json:"notes,omitempty"`
}

// RateKPIRequest represents request to rate KPI achievement
type RateKPIRequest struct {
	Rating int     `json:"rating" validate:"required,min=1,max=5"`
	Notes  *string `json:"notes,omitempty"`
}

// KPIFilter represents filter for KPIs
type KPIFilter struct {
	EmployeeID     *int64  `json:"employee_id,omitempty"`
	Year           *int    `json:"year,omitempty"`
	Month          *int    `json:"month,omitempty"`
	Quarter        *int    `json:"quarter,omitempty"`
	Category       *string `json:"category,omitempty"`
	DepartmentID   *int64  `json:"department_id,omitempty"`
	OrganizationID *int64  `json:"organization_id,omitempty"`
	Limit          int     `json:"limit,omitempty"`
	Offset         int     `json:"offset,omitempty"`
}

// BulkKPIRequest represents request to create multiple KPIs
type BulkKPIRequest struct {
	EmployeeID int64     `json:"employee_id" validate:"required"`
	Year       int       `json:"year" validate:"required"`
	KPIs       []KPIData `json:"kpis" validate:"required,min=1"`
}

// KPIData represents single KPI in bulk request
type KPIData struct {
	Name            string   `json:"name" validate:"required"`
	Description     *string  `json:"description,omitempty"`
	Category        *string  `json:"category,omitempty"`
	MeasurementUnit *string  `json:"measurement_unit,omitempty"`
	TargetValue     float64  `json:"target_value" validate:"required"`
	MinThreshold    *float64 `json:"min_threshold,omitempty"`
	MaxThreshold    *float64 `json:"max_threshold,omitempty"`
	Month           *int     `json:"month,omitempty"`
	Quarter         *int     `json:"quarter,omitempty"`
	Weight          float64  `json:"weight"`
}
