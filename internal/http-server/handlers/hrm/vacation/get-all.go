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

type AllGetter interface {
	GetAll(ctx context.Context, filters dto.VacationFilters) ([]*vacationmodel.Vacation, error)
}

func GetAll(log *slog.Logger, svc AllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filters dto.VacationFilters
		q := r.URL.Query()

		if v := q.Get("employee_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.EmployeeID = &val
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("vacation_type"); v != "" {
			filters.VacationType = &v
		}
		if v := q.Get("start_date"); v != "" {
			filters.StartDate = &v
		}
		if v := q.Get("end_date"); v != "" {
			filters.EndDate = &v
		}

		vacations, err := svc.GetAll(r.Context(), filters)
		if err != nil {
			log.Error("failed to get vacations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacations"))
			return
		}

		if vacations == nil {
			vacations = []*vacationmodel.Vacation{}
		}
		render.JSON(w, r, vacations)
	}
}
