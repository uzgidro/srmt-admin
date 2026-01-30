package dashboard

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
)

// DashboardRepository defines the interface for dashboard operations
type DashboardRepository interface {
	GetDashboard(ctx context.Context, filter hrm.DashboardFilter) (*hrmmodel.Dashboard, error)
}

// Get returns the HRM dashboard data
func Get(log *slog.Logger, repo DashboardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.dashboard.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Create empty filter - can be extended later to support query params
		var filter hrm.DashboardFilter

		dashboard, err := repo.GetDashboard(r.Context(), filter)
		if err != nil {
			log.Error("failed to get HRM dashboard", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve HRM dashboard"))
			return
		}

		log.Info("successfully retrieved HRM dashboard")
		render.JSON(w, r, dashboard)
	}
}
