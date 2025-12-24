package reservoirsummary

import (
	"context"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/dto"
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

type staticDataFetcher interface {
	FetchDataAtDayBegin(ctx context.Context, date string) (map[int64]*dto.OrganizationWithData, error)
}

// Get returns an HTTP handler that retrieves reservoir summary data
func Get(log *slog.Logger, getter reservoirSummaryGetter, fetcher staticDataFetcher) http.HandlerFunc {
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

		dataAtDayBegin, err := fetcher.FetchDataAtDayBegin(r.Context(), dateStr)
		if err != nil {
			log.Error("failed to fetch dataAtDayBegin", sl.Err(err))
		}

		for _, summary := range summaries {
			if summary.OrganizationID != nil {
				if val, ok := dataAtDayBegin[*summary.OrganizationID]; ok {
					isEdited := true
					if val.Data.Income != nil && *val.Data.Income != 0 && summary.Income.Current == 0 {
						summary.Income.Current = *val.Data.Income
						summary.Income.IsEdited = &isEdited
					}
					if val.Data.Release != nil && *val.Data.Release != 0 && summary.Release.Current == 0 {
						summary.Release.Current = *val.Data.Release
						summary.Release.IsEdited = &isEdited
					}
					if val.Data.Level != nil && *val.Data.Level != 0 && summary.Level.Current == 0 {
						summary.Level.Current = *val.Data.Level
						summary.Level.IsEdited = &isEdited
					}
					if val.Data.Volume != nil && *val.Data.Volume != 0 && summary.Volume.Current == 0 {
						summary.Volume.Current = *val.Data.Volume
						summary.Volume.IsEdited = &isEdited
					}
				}
			}
		}

		log.Info("successfully retrieved reservoir summaries",
			slog.Int("count", len(summaries)),
			slog.String("date", dateStr),
		)

		render.JSON(w, r, summaries)
	}
}
