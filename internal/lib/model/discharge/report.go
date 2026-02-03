package discharge

import "time"

// ReportRow - агрегированные данные по ГЭС для отчета холостых сбросов
type ReportRow struct {
	OrganizationID   int64
	OrganizationName string
	StartDate        time.Time  // min(StartedAt)
	StartTime        string     // HH:mm
	EndDate          *time.Time // max(EndedAt)
	EndTime          *string    // HH:mm
	Duration         string     // "X кун, Y соат, Z минут"
	TotalVolume      float64    // sum(TotalVolume)
	Reason           *string    // первая запись
}
