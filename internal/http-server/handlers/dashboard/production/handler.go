package production

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	gesproduction "srmt-admin/internal/lib/model/ges-production"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Provider interface {
	GetGesProductionDashboard(ctx context.Context) (*gesproduction.DashboardResponse, error)
}

func New(log *slog.Logger, provider Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.dashboard.production.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		data, err := provider.GetGesProductionDashboard(r.Context())
		if err != nil {
			log.Error("failed to get ges production dashboard", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get data"))
			return
		}

		// Handle empty data
		if data == nil {
			log.Info("no ges production data found")
			render.JSON(w, r, nil) // Or empty object? user asked for 0 if prev missing, but what if no data at all?
			// The repo returns nil if table is empty.
			// Let's return a default "zero" response or null.
			// Returning null is usually fine for "no data".
			return
		}

		render.JSON(w, r, data)
	}
}
