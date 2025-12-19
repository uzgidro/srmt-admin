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

type ReservoirFetcher interface {
	FetchHourly(ctx context.Context, date string) (map[int64][]*dto.ReservoirData, error)
}

func New(log *slog.Logger, reservoirFetcher ReservoirFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.dashboard.get-reservoir-hourly.New"
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

		// Fetch hourly reservoir data
		hourlyData, err := reservoirFetcher.FetchHourly(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to fetch hourly reservoir data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve hourly reservoir data"))
			return
		}

		log.Info("successfully retrieved hourly reservoir data", slog.Int("organizations", len(hourlyData)), slog.String("date", dateStr))
		render.JSON(w, r, hourlyData)
	}
}
