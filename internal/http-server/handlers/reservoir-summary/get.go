package reservoirsummary

import (
	"context"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/dto"
	"time"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// reservoirSummaryGetter defines the interface for retrieving reservoir
// summaries. It also provides level→volume curve lookups so the handler can
// recompute Volume.Current when Volume is missing but Level is known.
type reservoirSummaryGetter interface {
	GetReservoirSummary(ctx context.Context, date string) ([]*reservoirsummary.ResponseModel, error)
	volumeByLevelByOrg
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

		applyStaticFallbacks(r.Context(), log, summaries, dataAtDayBegin, getter)

		summaries = filterSummariesForCaller(r.Context(), summaries)

		log.Info("successfully retrieved reservoir summaries",
			slog.Int("count", len(summaries)),
			slog.String("date", dateStr),
		)

		render.JSON(w, r, summaries)
	}
}

// filterSummariesForCaller returns the rows visible to the current user.
// sc/rais see everything (incl. the ИТОГО row). Any other role gets only
// rows whose OrganizationID is in claims.OrganizationIDs; the ИТОГО row
// (OrganizationID == nil) is dropped — totals across the full report make
// no sense when the user only sees their own organization.
func filterSummariesForCaller(ctx context.Context, summaries []*reservoirsummary.ResponseModel) []*reservoirsummary.ResponseModel {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return []*reservoirsummary.ResponseModel{}
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return summaries
		}
	}
	if len(claims.OrganizationIDs) == 0 {
		return []*reservoirsummary.ResponseModel{}
	}
	filtered := make([]*reservoirsummary.ResponseModel, 0, 1)
	for _, s := range summaries {
		if s.OrganizationID != nil && auth.ContainsOrg(claims.OrganizationIDs, *s.OrganizationID) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
