package analytics

import (
	"net/http"
	"srmt-admin/internal/lib/dto"
	"strconv"
)

func parseReportFilter(r *http.Request) dto.ReportFilter {
	q := r.URL.Query()
	var filter dto.ReportFilter

	if v := q.Get("start_date"); v != "" {
		filter.StartDate = &v
	}
	if v := q.Get("end_date"); v != "" {
		filter.EndDate = &v
	}
	if v := q.Get("department_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.DepartmentID = &id
		}
	}
	if v := q.Get("position_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.PositionID = &id
		}
	}
	if v := q.Get("report_type"); v != "" {
		filter.ReportType = &v
	}

	return filter
}
