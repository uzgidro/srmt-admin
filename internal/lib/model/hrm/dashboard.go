package hrm

import "time"

// Dashboard represents HRM dashboard data
type Dashboard struct {
	// Employee stats
	EmployeeStats EmployeeStats `json:"employee_stats"`

	// Vacation stats
	VacationStats VacationStats `json:"vacation_stats"`

	// Recruiting stats
	RecruitingStats RecruitingStats `json:"recruiting_stats"`

	// Training stats
	TrainingStats TrainingStats `json:"training_stats"`

	// Performance stats
	PerformanceStats PerformanceStats `json:"performance_stats"`

	// Quick metrics
	PendingApprovals  int `json:"pending_approvals"`
	ExpiringDocuments int `json:"expiring_documents"`
	UpcomingReviews   int `json:"upcoming_reviews"`
	NewNotifications  int `json:"new_notifications"`

	// Birthdays this week
	UpcomingBirthdays []EmployeeBirthday `json:"upcoming_birthdays,omitempty"`

	// Recent hires
	RecentHires []Employee `json:"recent_hires,omitempty"`

	// Anniversaries this month
	Anniversaries []EmployeeAnniversary `json:"anniversaries,omitempty"`

	GeneratedAt time.Time `json:"generated_at"`
}

// EmployeeStats represents employee statistics
type EmployeeStats struct {
	TotalEmployees     int            `json:"total_employees"`
	ActiveEmployees    int            `json:"active_employees"`
	OnLeaveCount       int            `json:"on_leave_count"`
	TerminatedThisYear int            `json:"terminated_this_year"`
	HiredThisYear      int            `json:"hired_this_year"`
	TurnoverRate       float64        `json:"turnover_rate_percent"`
	ByDepartment       map[string]int `json:"by_department,omitempty"`
	ByEmploymentType   map[string]int `json:"by_employment_type,omitempty"`
}

// VacationStats represents vacation statistics
type VacationStats struct {
	PendingRequests     int     `json:"pending_requests"`
	ApprovedThisMonth   int     `json:"approved_this_month"`
	OnVacationToday     int     `json:"on_vacation_today"`
	AverageVacationDays float64 `json:"average_vacation_days_used"`
	EmployeesLowBalance int     `json:"employees_low_balance"` // < 5 days remaining
}

// EmployeeBirthday represents upcoming birthday
type EmployeeBirthday struct {
	EmployeeID   int64     `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	Department   string    `json:"department"`
	BirthDate    time.Time `json:"birth_date"`
	DaysUntil    int       `json:"days_until"`
}

// EmployeeAnniversary represents work anniversary
type EmployeeAnniversary struct {
	EmployeeID   int64     `json:"employee_id"`
	EmployeeName string    `json:"employee_name"`
	Department   string    `json:"department"`
	HireDate     time.Time `json:"hire_date"`
	YearsWorked  int       `json:"years_worked"`
	DaysUntil    int       `json:"days_until"`
}

// DashboardFilter represents filter for dashboard data
type DashboardFilter struct {
	OrganizationID *int64 `json:"organization_id,omitempty"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	Year           *int   `json:"year,omitempty"`
	Month          *int   `json:"month,omitempty"`
}

// HeadcountTrend represents headcount over time
type HeadcountTrend struct {
	Month          string `json:"month"` // YYYY-MM
	TotalEmployees int    `json:"total_employees"`
	Hired          int    `json:"hired"`
	Terminated     int    `json:"terminated"`
	NetChange      int    `json:"net_change"`
}

// DepartmentDistribution represents employees per department
type DepartmentDistribution struct {
	DepartmentID   int64   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	EmployeeCount  int     `json:"employee_count"`
	Percentage     float64 `json:"percentage"`
}

// AgeDistribution represents employee age distribution
type AgeDistribution struct {
	Range string `json:"range"` // "18-25", "26-35", etc.
	Count int    `json:"count"`
}

// TenureDistribution represents employee tenure distribution
type TenureDistribution struct {
	Range string `json:"range"` // "<1 year", "1-3 years", etc.
	Count int    `json:"count"`
}

// AnalyticsReport represents detailed analytics report
type AnalyticsReport struct {
	Period             string                   `json:"period"` // "2024-Q1", "2024-01", etc.
	HeadcountTrend     []HeadcountTrend         `json:"headcount_trend"`
	DepartmentDist     []DepartmentDistribution `json:"department_distribution"`
	AgeDistribution    []AgeDistribution        `json:"age_distribution"`
	TenureDistribution []TenureDistribution     `json:"tenure_distribution"`
	TurnoverAnalysis   TurnoverAnalysis         `json:"turnover_analysis"`
	SalaryAnalysis     SalaryAnalysis           `json:"salary_analysis"`
}

// TurnoverAnalysis represents turnover analysis
type TurnoverAnalysis struct {
	VoluntaryCount   int            `json:"voluntary_count"`
	InvoluntaryCount int            `json:"involuntary_count"`
	TotalCount       int            `json:"total_count"`
	TurnoverRate     float64        `json:"turnover_rate_percent"`
	TopReasons       map[string]int `json:"top_reasons,omitempty"`
	ByDepartment     map[string]int `json:"by_department,omitempty"`
}

// SalaryAnalysis represents salary analysis
type SalaryAnalysis struct {
	AverageSalary    float64            `json:"average_salary"`
	MedianSalary     float64            `json:"median_salary"`
	TotalPayroll     float64            `json:"total_payroll"`
	ByDepartment     map[string]float64 `json:"by_department,omitempty"`
	ByPosition       map[string]float64 `json:"by_position,omitempty"`
	SalaryGrowthRate float64            `json:"salary_growth_rate_percent"`
}

// MyDashboard represents employee's personal dashboard
type MyDashboard struct {
	// Profile summary
	Employee *Employee `json:"employee"`

	// Leave balance
	VacationBalances []VacationBalance `json:"vacation_balances"`
	PendingVacations []Vacation        `json:"pending_vacations"`

	// Current month
	CurrentTimesheet *Timesheet `json:"current_timesheet,omitempty"`

	// Training
	UpcomingTrainings    []Training    `json:"upcoming_trainings"`
	ExpiringCertificates []Certificate `json:"expiring_certificates"`

	// Performance
	CurrentReview   *PerformanceReview `json:"current_review,omitempty"`
	ActiveGoals     []PerformanceGoal  `json:"active_goals"`
	DevelopmentPlan *DevelopmentPlan   `json:"development_plan,omitempty"`

	// Pending tasks
	PendingDocuments   []Document             `json:"pending_documents"`
	PendingAssessments []CompetencyAssessment `json:"pending_assessments"`

	// Notifications
	UnreadNotifications int            `json:"unread_notifications"`
	RecentNotifications []Notification `json:"recent_notifications"`

	// Quick stats
	DaysUntilVacation *int `json:"days_until_vacation,omitempty"`
	YearsWorked       int  `json:"years_worked"`

	GeneratedAt time.Time `json:"generated_at"`
}
