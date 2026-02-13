package performance

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/performance"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DashboardGetter interface {
	GetPerformanceDashboard(ctx context.Context) (*performance.PerformanceDashboard, error)
}

func GetDashboard(log *slog.Logger, svc DashboardGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetDashboard"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		result, err := svc.GetPerformanceDashboard(r.Context())
		if err != nil {
			log.Error("failed to get performance dashboard", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve performance dashboard"))
			return
		}

		render.JSON(w, r, result)
	}
}
