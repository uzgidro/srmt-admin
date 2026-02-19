package reservoirsummaryhourly

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/reservoir-hourly"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// ReportBuilder builds the hourly reservoir report
type ReportBuilder interface {
	BuildReport(ctx context.Context, date string) (*model.HourlyReport, error)
}

// GetExport returns an HTTP handler that builds and returns the hourly reservoir report as JSON
func GetExport(log *slog.Logger, builder ReportBuilder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirsummaryhourly.GetExport"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse date query parameter, default to today
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		// Validate date format
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			log.Warn("invalid date format", sl.Err(err), slog.String("date", dateStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid date format. Expected YYYY-MM-DD"))
			return
		}

		// Build report
		report, err := builder.BuildReport(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to build hourly report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to build hourly report"))
			return
		}

		render.JSON(w, r, report)
	}
}
