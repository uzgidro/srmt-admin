package salary

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/salary"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllGetter interface {
	GetAll(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error)
}

func GetAll(log *slog.Logger, svc AllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.SalaryFilters

		if v := q.Get("period_year"); v != "" {
			val, _ := strconv.Atoi(v)
			filters.PeriodYear = &val
		}
		if v := q.Get("period_month"); v != "" {
			val, _ := strconv.Atoi(v)
			filters.PeriodMonth = &val
		}
		if v := q.Get("department_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.DepartmentID = &val
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}

		salaries, err := svc.GetAll(r.Context(), filters)
		if err != nil {
			log.Error("failed to get salaries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salaries"))
			return
		}

		render.JSON(w, r, salaries)
	}
}
