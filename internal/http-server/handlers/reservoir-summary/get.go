package reservoirsummary

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// reservoirSummaryGetter defines the interface for retrieving reservoir summaries
type reservoirSummaryGetter interface {
	GetReservoirSummary(ctx context.Context, date string) ([]*reservoirsummary.ResponseModel, error)
}

// Get returns an HTTP handler that retrieves reservoir summary data
func Get(log *slog.Logger, getter reservoirSummaryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummary.Get"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse and validate date query parameter
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			log.Warn("missing required 'date' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'date' parameter (format: YYYY-MM-DD)"))
			return
		}

		// Validate date format (YYYY-MM-DD)
		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Warn("invalid date format", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
			return
		}

		// Retrieve reservoir summary data
		summaries, err := getter.GetReservoirSummary(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to get reservoir summaries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve reservoir summaries"))
			return
		}

		log.Info("successfully retrieved reservoir summaries",
			slog.Int("count", len(summaries)),
			slog.String("date", dateStr),
		)

		render.JSON(w, r, summaries)
	}
}
