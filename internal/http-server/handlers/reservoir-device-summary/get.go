package reservoirdevicesummary

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	reservoirdevicesummary "srmt-admin/internal/lib/model/reservoir-device-summary"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type reservoirDeviceSummaryGetter interface {
	GetReservoirDeviceSummary(ctx context.Context, date *time.Time) ([]*reservoirdevicesummary.ResponseModel, error)
}

func Get(log *slog.Logger, getter reservoirDeviceSummaryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirdevicesummary.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var date *time.Time
		if dateStr := r.URL.Query().Get("date"); dateStr != "" {
			parsed, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Warn("invalid date format", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
				return
			}
			// End of the given day
			eod := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999999999, time.UTC)
			date = &eod
		}

		summaries, err := getter.GetReservoirDeviceSummary(r.Context(), date)
		if err != nil {
			log.Error("failed to get reservoir device summaries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve reservoir device summaries"))
			return
		}

		log.Info("successfully retrieved reservoir device summaries", slog.Int("count", len(summaries)))
		render.JSON(w, r, summaries)
	}
}
