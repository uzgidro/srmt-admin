package analytics

// DistributionItem — универсальный элемент распределения
type DistributionItem struct {
	Label      string  `json:"label"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// TrendPoint — точка на графике тренда
type TrendPoint struct {
	Year  int     `json:"year"`
	Month int     `json:"month"`
	Value float64 `json:"value"`
}

// Dashboard — главный аналитический дашборд
type Dashboard struct {
	TotalEmployees      int                    `json:"total_employees"`
	NewHiresMonth       int                    `json:"new_hires_month"`
	TerminationsMonth   int                    `json:"terminations_month"`
	TurnoverRate        float64                `json:"turnover_rate"`
	AvgTenureYears      float64                `json:"avg_tenure_years"`
	AvgAge              float64                `json:"avg_age"`
	GenderDistribution  *GenderDistribution    `json:"gender_distribution"`
	AgeDistribution     []*DistributionItem    `json:"age_distribution"`
	TenureDistribution  []*DistributionItem    `json:"tenure_distribution"`
	DepartmentHeadcount []*DepartmentHeadcount `json:"department_headcount"`
	PositionHeadcount   []*PositionHeadcount   `json:"position_headcount"`
}

type GenderDistribution struct {
	Male   int `json:"male"`
	Female int `json:"female"`
	Total  int `json:"total"`
}

type DepartmentHeadcount struct {
	DepartmentID   int64  `json:"department_id"`
	DepartmentName string `json:"department_name"`
	Headcount      int    `json:"headcount"`
}

type PositionHeadcount struct {
	PositionID   int64  `json:"position_id"`
	PositionName string `json:"position_name"`
	Count        int    `json:"count"`
}

// HeadcountReport
type HeadcountReport struct {
	TotalEmployees int                    `json:"total_employees"`
	ByDepartment   []*DepartmentHeadcount `json:"by_department"`
	ByPosition     []*PositionHeadcount   `json:"by_position"`
}

// HeadcountTrend
type HeadcountTrend struct {
	Points []*TrendPoint `json:"points"`
}

// TurnoverReport
type TurnoverReport struct {
	PeriodStart             string                `json:"period_start"`
	PeriodEnd               string                `json:"period_end"`
	TotalTerminations       int                   `json:"total_terminations"`
	VoluntaryTerminations   int                   `json:"voluntary_terminations"`
	InvoluntaryTerminations int                   `json:"involuntary_terminations"`
	TurnoverRate            float64               `json:"turnover_rate"`
	RetentionRate           float64               `json:"retention_rate"`
	AvgTenureAtTermination  float64               `json:"avg_tenure_at_termination"`
	ByReason                []*DistributionItem   `json:"by_reason"`
	ByDepartment            []*DepartmentTurnover `json:"by_department"`
}

type DepartmentTurnover struct {
	Department   string  `json:"department"`
	Terminations int     `json:"terminations"`
	TurnoverRate float64 `json:"turnover_rate"`
}

// TurnoverTrend
type TurnoverTrend struct {
	Points []*TrendPoint `json:"points"`
}

// AttendanceReport
type AttendanceReport struct {
	PeriodStart   string                  `json:"period_start"`
	PeriodEnd     string                  `json:"period_end"`
	TotalWorkDays int                     `json:"total_work_days"`
	AvgAttendance float64                 `json:"avg_attendance"`
	AvgAbsence    float64                 `json:"avg_absence"`
	ByStatus      []*DistributionItem     `json:"by_status"`
	ByDepartment  []*DepartmentAttendance `json:"by_department"`
}

type DepartmentAttendance struct {
	Department     string  `json:"department"`
	AttendanceRate float64 `json:"attendance_rate"`
	AbsenceRate    float64 `json:"absence_rate"`
}

// SalaryReport
type SalaryReport struct {
	PeriodStart  string              `json:"period_start"`
	PeriodEnd    string              `json:"period_end"`
	TotalPayroll float64             `json:"total_payroll"`
	AvgSalary    float64             `json:"avg_salary"`
	MedianSalary float64             `json:"median_salary"`
	MinSalary    float64             `json:"min_salary"`
	MaxSalary    float64             `json:"max_salary"`
	ByDepartment []*DepartmentSalary `json:"by_department"`
}

type DepartmentSalary struct {
	Department   string  `json:"department"`
	AvgSalary    float64 `json:"avg_salary"`
	TotalPayroll float64 `json:"total_payroll"`
	Headcount    int     `json:"headcount"`
}

// SalaryTrend
type SalaryTrend struct {
	Points []*TrendPoint `json:"points"`
}

// PerformanceAnalytics
type PerformanceAnalytics struct {
	TotalReviews       int                      `json:"total_reviews"`
	AvgRating          float64                  `json:"avg_rating"`
	RatingDistribution []*DistributionItem      `json:"rating_distribution"`
	GoalCompletion     *GoalCompletion          `json:"goal_completion"`
	ByDepartment       []*DepartmentPerformance `json:"by_department"`
}

type GoalCompletion struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Rate      float64 `json:"rate"`
}

type DepartmentPerformance struct {
	Department string  `json:"department"`
	AvgRating  float64 `json:"avg_rating"`
	GoalRate   float64 `json:"goal_rate"`
}

// TrainingAnalytics
type TrainingAnalytics struct {
	TotalTrainings    int                 `json:"total_trainings"`
	TotalParticipants int                 `json:"total_participants"`
	CompletionRate    float64             `json:"completion_rate"`
	ByStatus          []*DistributionItem `json:"by_status"`
	ByType            []*DistributionItem `json:"by_type"`
}

// DemographicsReport
type DemographicsReport struct {
	TotalEmployees     int                 `json:"total_employees"`
	AvgAge             float64             `json:"avg_age"`
	AgeDistribution    []*DistributionItem `json:"age_distribution"`
	TenureDistribution []*DistributionItem `json:"tenure_distribution"`
}

// DiversityReport — заглушка
type DiversityReport struct {
	Message string `json:"message"`
}
