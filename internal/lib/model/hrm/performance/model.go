package performance

import "time"

type PerformanceReview struct {
	ID             int64              `json:"id"`
	EmployeeID     int64              `json:"employee_id"`
	EmployeeName   string             `json:"employee_name"`
	ReviewerID     *int64             `json:"reviewer_id,omitempty"`
	ReviewerName   string             `json:"reviewer_name"`
	Type           string             `json:"type"`
	Status         string             `json:"status"`
	PeriodStart    string             `json:"period_start"`
	PeriodEnd      string             `json:"period_end"`
	Goals          []*PerformanceGoal `json:"goals"`
	SelfRating     *int               `json:"self_rating,omitempty"`
	ManagerRating  *int               `json:"manager_rating,omitempty"`
	FinalRating    *int               `json:"final_rating,omitempty"`
	SelfComment    *string            `json:"self_comment,omitempty"`
	ManagerComment *string            `json:"manager_comment,omitempty"`
	Strengths      *string            `json:"strengths,omitempty"`
	Improvements   *string            `json:"improvements,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type PerformanceGoal struct {
	ID           int64     `json:"id"`
	ReviewID     *int64    `json:"review_id,omitempty"`
	EmployeeID   int64     `json:"employee_id"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	Metric       *string   `json:"metric,omitempty"`
	TargetValue  float64   `json:"target_value"`
	CurrentValue float64   `json:"current_value"`
	Weight       float64   `json:"weight"`
	Status       string    `json:"status"`
	DueDate      string    `json:"due_date"`
	Progress     int       `json:"progress"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type KPI struct {
	EmployeeID   int64   `json:"employee_id"`
	EmployeeName string  `json:"employee_name"`
	Department   string  `json:"department"`
	Position     string  `json:"position"`
	GoalsTotal   int     `json:"goals_total"`
	GoalsDone    int     `json:"goals_done"`
	AvgProgress  float64 `json:"avg_progress"`
	AvgRating    float64 `json:"avg_rating"`
}

type EmployeeRating struct {
	EmployeeID     int64                   `json:"employee_id"`
	EmployeeName   string                  `json:"employee_name"`
	Department     string                  `json:"department"`
	Position       string                  `json:"position"`
	ReviewsCount   int                     `json:"reviews_count"`
	AvgFinalRating float64                 `json:"avg_final_rating"`
	Details        []*EmployeeRatingDetail `json:"details"`
}

type EmployeeRatingDetail struct {
	ReviewID    int64  `json:"review_id"`
	Type        string `json:"type"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	FinalRating *int   `json:"final_rating,omitempty"`
	Status      string `json:"status"`
}

type PerformanceDashboard struct {
	TotalReviews     int        `json:"total_reviews"`
	CompletedReviews int        `json:"completed_reviews"`
	PendingReviews   int        `json:"pending_reviews"`
	AvgRating        float64    `json:"avg_rating"`
	GoalStats        *GoalStats `json:"goal_stats"`
}

type GoalStats struct {
	Total       int     `json:"total"`
	Completed   int     `json:"completed"`
	InProgress  int     `json:"in_progress"`
	Overdue     int     `json:"overdue"`
	AvgProgress float64 `json:"avg_progress"`
}
