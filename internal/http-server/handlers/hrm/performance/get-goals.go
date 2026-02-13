package performance

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/performance"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type GoalAllGetter interface {
	GetAllGoals(ctx context.Context, filters dto.GoalFilters) ([]*performance.PerformanceGoal, error)
}

func GetGoals(log *slog.Logger, svc GoalAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetGoals"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.GoalFilters

		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("employee_id"); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				filters.EmployeeID = &id
			}
		}
		if v := q.Get("review_id"); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				filters.ReviewID = &id
			}
		}

		result, err := svc.GetAllGoals(r.Context(), filters)
		if err != nil {
			log.Error("failed to get goals", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve goals"))
			return
		}

		render.JSON(w, r, result)
	}
}
