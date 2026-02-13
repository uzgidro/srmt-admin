package timesheet

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	ts "srmt-admin/internal/lib/model/hrm/timesheet"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type HolidayGetter interface {
	GetHolidays(ctx context.Context, year int) ([]*ts.Holiday, error)
}

func GetHolidays(log *slog.Logger, svc HolidayGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.GetHolidays"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		year, err := strconv.Atoi(r.URL.Query().Get("year"))
		if err != nil || year < 2000 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Valid 'year' query parameter is required"))
			return
		}

		holidays, err := svc.GetHolidays(r.Context(), year)
		if err != nil {
			log.Error("failed to get holidays", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve holidays"))
			return
		}

		render.JSON(w, r, holidays)
	}
}
