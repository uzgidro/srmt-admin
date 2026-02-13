package dto

// --- Reviews ---

type CreateReviewRequest struct {
	EmployeeID  int64  `json:"employee_id" validate:"required"`
	ReviewerID  *int64 `json:"reviewer_id,omitempty"`
	Type        string `json:"type" validate:"required,oneof=annual quarterly probation project mid_year"`
	PeriodStart string `json:"period_start" validate:"required"`
	PeriodEnd   string `json:"period_end" validate:"required"`
}

type UpdateReviewRequest struct {
	Type        *string `json:"type,omitempty" validate:"omitempty,oneof=annual quarterly probation project mid_year"`
	PeriodStart *string `json:"period_start,omitempty"`
	PeriodEnd   *string `json:"period_end,omitempty"`
	ReviewerID  *int64  `json:"reviewer_id,omitempty"`
}

type SelfReviewRequest struct {
	SelfRating  int     `json:"self_rating" validate:"required,min=1,max=5"`
	SelfComment *string `json:"self_comment,omitempty"`
}

type ManagerReviewRequest struct {
	ManagerRating  int     `json:"manager_rating" validate:"required,min=1,max=5"`
	ManagerComment *string `json:"manager_comment,omitempty"`
	FinalRating    *int    `json:"final_rating,omitempty" validate:"omitempty,min=1,max=5"`
	Strengths      *string `json:"strengths,omitempty"`
	Improvements   *string `json:"improvements,omitempty"`
}

type ReviewFilters struct {
	Status     *string
	Type       *string
	EmployeeID *int64
	Search     *string
}

// --- Goals ---

type CreateGoalRequest struct {
	ReviewID    *int64   `json:"review_id,omitempty"`
	EmployeeID  int64    `json:"employee_id" validate:"required"`
	Title       string   `json:"title" validate:"required"`
	Description *string  `json:"description,omitempty"`
	Metric      *string  `json:"metric,omitempty"`
	TargetValue *float64 `json:"target_value,omitempty"`
	Weight      *float64 `json:"weight,omitempty"`
	DueDate     string   `json:"due_date" validate:"required"`
}

type UpdateGoalRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Metric      *string  `json:"metric,omitempty"`
	TargetValue *float64 `json:"target_value,omitempty"`
	Weight      *float64 `json:"weight,omitempty"`
	Status      *string  `json:"status,omitempty" validate:"omitempty,oneof=not_started in_progress completed overdue cancelled"`
	DueDate     *string  `json:"due_date,omitempty"`
}

type UpdateGoalProgressRequest struct {
	CurrentValue *float64 `json:"current_value,omitempty"`
	Progress     *int     `json:"progress,omitempty" validate:"omitempty,min=0,max=100"`
}

type GoalFilters struct {
	Status     *string
	EmployeeID *int64
	ReviewID   *int64
}
