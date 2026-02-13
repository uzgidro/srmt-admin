package dto

type ReportFilter struct {
	StartDate    *string
	EndDate      *string
	DepartmentID *int64
	PositionID   *int64
	ReportType   *string
}
