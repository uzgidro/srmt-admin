package hrm

import "time"

// --- Dashboard DTOs ---

// DashboardFilter represents filter for dashboard data
type DashboardFilter struct {
	OrganizationID *int64 `json:"organization_id,omitempty"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	Year           *int   `json:"year,omitempty"`
	Month          *int   `json:"month,omitempty"`
}

// MyDashboardFilter represents filter for employee's personal dashboard
type MyDashboardFilter struct {
	Year  *int `json:"year,omitempty"`
	Month *int `json:"month,omitempty"`
}

// --- Analytics DTOs ---

// AnalyticsFilter represents filter for analytics reports
type AnalyticsFilter struct {
	OrganizationID *int64    `json:"organization_id,omitempty"`
	DepartmentID   *int64    `json:"department_id,omitempty"`
	FromDate       time.Time `json:"from_date" validate:"required"`
	ToDate         time.Time `json:"to_date" validate:"required"`
	GroupBy        string    `json:"group_by" validate:"omitempty,oneof=month quarter year"`
}

// HeadcountReportFilter represents filter for headcount report
type HeadcountReportFilter struct {
	OrganizationID *int64    `json:"organization_id,omitempty"`
	DepartmentID   *int64    `json:"department_id,omitempty"`
	FromDate       time.Time `json:"from_date" validate:"required"`
	ToDate         time.Time `json:"to_date" validate:"required"`
}

// TurnoverReportFilter represents filter for turnover report
type TurnoverReportFilter struct {
	OrganizationID *int64 `json:"organization_id,omitempty"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	Year           int    `json:"year" validate:"required"`
}

// SalaryReportFilter represents filter for salary report
type SalaryReportFilter struct {
	OrganizationID *int64 `json:"organization_id,omitempty"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	Year           int    `json:"year" validate:"required"`
	Month          *int   `json:"month,omitempty"`
}

// VacationReportFilter represents filter for vacation report
type VacationReportFilter struct {
	OrganizationID *int64 `json:"organization_id,omitempty"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	Year           int    `json:"year" validate:"required"`
}

// TrainingReportFilter represents filter for training report
type TrainingReportFilter struct {
	OrganizationID *int64    `json:"organization_id,omitempty"`
	DepartmentID   *int64    `json:"department_id,omitempty"`
	FromDate       time.Time `json:"from_date" validate:"required"`
	ToDate         time.Time `json:"to_date" validate:"required"`
	TrainingType   *string   `json:"training_type,omitempty"`
}

// PerformanceReportFilter represents filter for performance report
type PerformanceReportFilter struct {
	OrganizationID *int64  `json:"organization_id,omitempty"`
	DepartmentID   *int64  `json:"department_id,omitempty"`
	Year           int     `json:"year" validate:"required"`
	ReviewType     *string `json:"review_type,omitempty"`
}

// --- Export DTOs ---

// ExportRequest represents request to export data
type ExportRequest struct {
	Format     string `json:"format" validate:"required,oneof=pdf excel csv"`
	ReportType string `json:"report_type" validate:"required"`
}

// EmployeeExportFilter represents filter for employee export
type EmployeeExportFilter struct {
	OrganizationID    *int64   `json:"organization_id,omitempty"`
	DepartmentID      *int64   `json:"department_id,omitempty"`
	EmploymentStatus  *string  `json:"employment_status,omitempty"`
	IncludeTerminated bool     `json:"include_terminated"`
	Fields            []string `json:"fields,omitempty"` // Specific fields to include
}

// VacationExportFilter represents filter for vacation export
type VacationExportFilter struct {
	Year           int    `json:"year" validate:"required"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	OrganizationID *int64 `json:"organization_id,omitempty"`
	VacationTypeID *int   `json:"vacation_type_id,omitempty"`
}

// SalaryExportFilter represents filter for salary export
type SalaryExportFilter struct {
	Year           int    `json:"year" validate:"required"`
	Month          *int   `json:"month,omitempty"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	OrganizationID *int64 `json:"organization_id,omitempty"`
}

// TimesheetExportFilter represents filter for timesheet export
type TimesheetExportFilter struct {
	Year           int    `json:"year" validate:"required"`
	Month          int    `json:"month" validate:"required,min=1,max=12"`
	DepartmentID   *int64 `json:"department_id,omitempty"`
	OrganizationID *int64 `json:"organization_id,omitempty"`
	EmployeeID     *int64 `json:"employee_id,omitempty"`
}
