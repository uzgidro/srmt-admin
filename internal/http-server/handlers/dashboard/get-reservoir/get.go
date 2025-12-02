package get

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ReservoirGetter interface {
	GetOrganizationsWithReservoir(ctx context.Context, orgIDs []int64, reservoirFetcher dto.ReservoirFetcher, date string) ([]*dto.OrganizationWithReservoir, error)
}

func New(log *slog.Logger, getter ReservoirGetter, reservoirFetcher dto.ReservoirFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.dashboard.get-reservoir.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse date parameter, default to today if not provided
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		// Validate date format
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			log.Error("invalid date format", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format, expected yyyy-MM-dd"))
			return
		}

		// Get organizations with reservoir metrics
		organizations, err := getter.GetOrganizationsWithReservoir(r.Context(), reservoirFetcher.GetIDs(), reservoirFetcher, dateStr)
		if err != nil {
			log.Error("failed to get organizations with reservoir data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve reservoir data"))
			return
		}

		log.Info("successfully retrieved organizations with reservoir data", slog.Int("count", len(organizations)), slog.String("date", dateStr))
		render.JSON(w, r, organizations)
	}
}
