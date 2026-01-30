package analytics

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
)

// DashboardRepository defines the interface for dashboard operations
type DashboardRepository interface {
	GetDashboardStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.DashboardResponse, error)
}

// GetDashboard returns the HRM analytics dashboard
func GetDashboard(log *slog.Logger, repo DashboardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetDashboard"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse optional filters from query params
		filter := hrm.AnalyticsFilter{}
		// Could parse organization_id, department_id from query params

		// Get dashboard data
		dashboard, err := repo.GetDashboardStats(r.Context(), filter)
		if err != nil {
			log.Error("failed to get dashboard stats", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve dashboard data"))
			return
		}

		log.Info("dashboard retrieved")
		render.JSON(w, r, dashboard)
	}
}
