package timesheet

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	ts "srmt-admin/internal/lib/model/hrm/timesheet"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type TimesheetGetter interface {
	GetTimesheet(ctx context.Context, filters dto.TimesheetFilters) ([]*ts.EmployeeTimesheet, error)
}

func GetAll(log *slog.Logger, svc TimesheetGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()

		year, err := strconv.Atoi(q.Get("year"))
		if err != nil || year < 2000 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Valid 'year' query parameter is required"))
			return
		}

		month, err := strconv.Atoi(q.Get("month"))
		if err != nil || month < 1 || month > 12 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Valid 'month' query parameter is required (1-12)"))
			return
		}

		filters := dto.TimesheetFilters{
			Year:  year,
			Month: month,
		}

		if v := q.Get("department_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.DepartmentID = &val
		}
		if v := q.Get("employee_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.EmployeeID = &val
		}

		timesheets, err := svc.GetTimesheet(r.Context(), filters)
		if err != nil {
			log.Error("failed to get timesheets", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve timesheets"))
			return
		}

		render.JSON(w, r, timesheets)
	}
}
