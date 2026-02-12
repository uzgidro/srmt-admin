package vacation

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	vacationmodel "srmt-admin/internal/lib/model/hrm/vacation"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type CalendarGetter interface {
	GetCalendar(ctx context.Context, filters dto.VacationCalendarFilters) ([]*vacationmodel.CalendarEntry, error)
}

func GetCalendar(log *slog.Logger, svc CalendarGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetCalendar"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filters dto.VacationCalendarFilters
		q := r.URL.Query()

		if v := q.Get("department_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.DepartmentID = &val
		}
		if v := q.Get("start_date"); v != "" {
			filters.StartDate = &v
		}
		if v := q.Get("end_date"); v != "" {
			filters.EndDate = &v
		}

		entries, err := svc.GetCalendar(r.Context(), filters)
		if err != nil {
			log.Error("failed to get vacation calendar", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get calendar"))
			return
		}

		if entries == nil {
			entries = []*vacationmodel.CalendarEntry{}
		}
		render.JSON(w, r, entries)
	}
}
