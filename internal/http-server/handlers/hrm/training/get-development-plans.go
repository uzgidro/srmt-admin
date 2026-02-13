package training

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/training"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DevelopmentPlansGetter interface {
	GetAllDevelopmentPlans(ctx context.Context, employeeID *int64) ([]*training.DevelopmentPlan, error)
}

func GetDevelopmentPlans(log *slog.Logger, svc DevelopmentPlansGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetDevelopmentPlans"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var employeeID *int64
		if v := r.URL.Query().Get("employee_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			employeeID = &val
		}

		plans, err := svc.GetAllDevelopmentPlans(r.Context(), employeeID)
		if err != nil {
			log.Error("failed to get development plans", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve development plans"))
			return
		}

		render.JSON(w, r, plans)
	}
}
