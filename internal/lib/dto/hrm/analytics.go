package hrm

import "time"

// --- Dashboard Response DTOs ---

// DashboardResponse represents the HRM dashboard data
type DashboardResponse struct {
	TotalEmployees        int                   `json:"total_employees"`
	ActiveEmployees       int                   `json:"active_employees"`
	NewHiresThisMonth     int                   `json:"new_hires_this_month"`
	TerminationsThisMonth int                   `json:"terminations_this_month"`
	PendingVacations      int                   `json:"pending_vacations"`
	PendingApprovals      int                   `json:"pending_approvals"`
	HeadcountByDepartment []DepartmentHeadcount `json:"headcount_by_department"`
	RecentActivity        []ActivityItem        `json:"recent_activity"`
}

// DepartmentHeadcount represents headcount per department
type DepartmentHeadcount struct {
	DepartmentID   int64  `json:"department_id"`
	DepartmentName string `json:"department_name"`
	Count          int    `json:"count"`
	ActiveCount    int    `json:"active_count"`
}

// ActivityItem represents recent activity item
type ActivityItem struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	EntityID    int64     `json:"entity_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// --- Headcount Report DTOs ---

// HeadcountReportResponse represents headcount report
type HeadcountReportResponse struct {
	TotalHeadcount     int                   `json:"total_headcount"`
	ActiveHeadcount    int                   `json:"active_headcount"`
	ByDepartment       []DepartmentHeadcount `json:"by_department"`
	ByEmploymentType   []TypeHeadcount       `json:"by_employment_type"`
	ByEmploymentStatus []StatusHeadcount     `json:"by_employment_status"`
}

// HeadcountTrendResponse represents headcount trend over time
type HeadcountTrendResponse struct {
	DataPoints []HeadcountTrendPoint `json:"data_points"`
}

// HeadcountTrendPoint represents a single data point in trend
type HeadcountTrendPoint struct {
	Date       string `json:"date"`
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	Headcount  int    `json:"headcount"`
	Hired      int    `json:"hired"`
	Terminated int    `json:"terminated"`
}

// TypeHeadcount represents headcount by type
type TypeHeadcount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// StatusHeadcount represents headcount by status
type StatusHeadcount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// --- Turnover Report DTOs ---

// TurnoverReportResponse represents turnover report
type TurnoverReportResponse struct {
	Period         string               `json:"period"`
	TotalEmployees int                  `json:"total_employees"`
	Hired          int                  `json:"hired"`
	Terminated     int                  `json:"terminated"`
	TurnoverRate   float64              `json:"turnover_rate"`
	RetentionRate  float64              `json:"retention_rate"`
	ByDepartment   []DepartmentTurnover `json:"by_department"`
	ByReason       []ReasonCount        `json:"by_reason,omitempty"`
}

// TurnoverTrendResponse represents turnover trend
type TurnoverTrendResponse struct {
	DataPoints []TurnoverTrendPoint `json:"data_points"`
}

// TurnoverTrendPoint represents a single data point
type TurnoverTrendPoint struct {
	Date         string  `json:"date"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
	TurnoverRate float64 `json:"turnover_rate"`
	Hired        int     `json:"hired"`
	Terminated   int     `json:"terminated"`
}

// DepartmentTurnover represents turnover by department
type DepartmentTurnover struct {
	DepartmentID   int64   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	Hired          int     `json:"hired"`
	Terminated     int     `json:"terminated"`
	TurnoverRate   float64 `json:"turnover_rate"`
}

// ReasonCount represents count by reason
type ReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// --- Attendance Report DTOs ---

// AttendanceReportResponse represents attendance report
type AttendanceReportResponse struct {
	Period             string                 `json:"period"`
	TotalWorkDays      int                    `json:"total_work_days"`
	AverageAttendance  float64                `json:"average_attendance"`
	TotalAbsences      int                    `json:"total_absences"`
	TotalVacationDays  int                    `json:"total_vacation_days"`
	TotalSickDays      int                    `json:"total_sick_days"`
	TotalOvertimeHours float64                `json:"total_overtime_hours"`
	ByDepartment       []DepartmentAttendance `json:"by_department"`
}

// DepartmentAttendance represents attendance by department
type DepartmentAttendance struct {
	DepartmentID   int64   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	AttendanceRate float64 `json:"attendance_rate"`
	AbsenceDays    int     `json:"absence_days"`
	OvertimeHours  float64 `json:"overtime_hours"`
}

// --- Salary Report DTOs ---

// SalaryReportResponse represents salary report
type SalaryReportResponse struct {
	Period        string             `json:"period"`
	TotalGross    float64            `json:"total_gross"`
	TotalNet      float64            `json:"total_net"`
	TotalTax      float64            `json:"total_tax"`
	TotalBonuses  float64            `json:"total_bonuses"`
	AverageSalary float64            `json:"average_salary"`
	MedianSalary  float64            `json:"median_salary"`
	ByDepartment  []DepartmentSalary `json:"by_department"`
}

// SalaryTrendResponse represents salary trend
type SalaryTrendResponse struct {
	DataPoints []SalaryTrendPoint `json:"data_points"`
}

// SalaryTrendPoint represents a single salary data point
type SalaryTrendPoint struct {
	Date          string  `json:"date"`
	Year          int     `json:"year"`
	Month         int     `json:"month"`
	TotalGross    float64 `json:"total_gross"`
	TotalNet      float64 `json:"total_net"`
	AverageSalary float64 `json:"average_salary"`
}

// DepartmentSalary represents salary by department
type DepartmentSalary struct {
	DepartmentID   int64   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	TotalGross     float64 `json:"total_gross"`
	AverageSalary  float64 `json:"average_salary"`
	EmployeeCount  int     `json:"employee_count"`
}

// --- Performance Report DTOs ---

// PerformanceReportResponse represents performance report
type PerformanceReportResponse struct {
	Period             string                  `json:"period"`
	TotalReviews       int                     `json:"total_reviews"`
	CompletedReviews   int                     `json:"completed_reviews"`
	AverageRating      float64                 `json:"average_rating"`
	ByDepartment       []DepartmentPerformance `json:"by_department"`
	RatingDistribution []RatingCount           `json:"rating_distribution"`
}

// DepartmentPerformance represents performance by department
type DepartmentPerformance struct {
	DepartmentID   int64   `json:"department_id"`
	DepartmentName string  `json:"department_name"`
	AverageRating  float64 `json:"average_rating"`
	ReviewCount    int     `json:"review_count"`
}

// RatingCount represents count by rating
type RatingCount struct {
	Rating int `json:"rating"`
	Count  int `json:"count"`
}

// --- Training Report DTOs ---

// TrainingReportResponse represents training report
type TrainingReportResponse struct {
	Period                string             `json:"period"`
	TotalTrainings        int                `json:"total_trainings"`
	CompletedTrainings    int                `json:"completed_trainings"`
	TotalParticipants     int                `json:"total_participants"`
	AverageCompletionRate float64            `json:"average_completion_rate"`
	TotalTrainingHours    float64            `json:"total_training_hours"`
	TotalCost             float64            `json:"total_cost"`
	ByCategory            []CategoryTraining `json:"by_category"`
}

// CategoryTraining represents training by category
type CategoryTraining struct {
	Category       string  `json:"category"`
	TrainingCount  int     `json:"training_count"`
	Participants   int     `json:"participants"`
	CompletionRate float64 `json:"completion_rate"`
}

// --- Demographics Report DTOs ---

// DemographicsReportResponse represents demographics report
type DemographicsReportResponse struct {
	TotalEmployees     int           `json:"total_employees"`
	AverageAge         float64       `json:"average_age"`
	AverageTenure      float64       `json:"average_tenure_years"`
	AgeDistribution    []AgeGroup    `json:"age_distribution"`
	GenderDistribution []GenderCount `json:"gender_distribution"`
	TenureDistribution []TenureGroup `json:"tenure_distribution"`
}

// AgeGroup represents age group count
type AgeGroup struct {
	AgeRange string  `json:"age_range"`
	Count    int     `json:"count"`
	Percent  float64 `json:"percent"`
}

// GenderCount represents gender count
type GenderCount struct {
	Gender  string  `json:"gender"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// TenureGroup represents tenure group count
type TenureGroup struct {
	TenureRange string  `json:"tenure_range"`
	Count       int     `json:"count"`
	Percent     float64 `json:"percent"`
}

// --- Export DTOs ---

// ExportResponse represents export response
type ExportResponse struct {
	FileID   int64  `json:"file_id"`
	FileName string `json:"file_name"`
	FileURL  string `json:"file_url,omitempty"`
}

// --- Custom Report DTOs ---

// CustomReportRequest represents custom report request
type CustomReportRequest struct {
	Metrics   []string        `json:"metrics" validate:"required,min=1"`
	GroupBy   []string        `json:"group_by"`
	Filter    AnalyticsFilter `json:"filter"`
	SortBy    *string         `json:"sort_by,omitempty"`
	SortOrder *string         `json:"sort_order,omitempty"`
	Limit     int             `json:"limit,omitempty"`
}

// CustomReportResponse represents custom report response
type CustomReportResponse struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Total   int                      `json:"total"`
}
