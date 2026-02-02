package askue

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
)

// ASCUEMetricsGetter defines the interface for fetching ASCUE metrics
type ASCUEMetricsGetter interface {
	FetchAll(ctx context.Context) (map[int64]*dto.ASCUEMetrics, error)
}

// New creates a handler for GET /ges/{id}/askue
func New(log *slog.Logger, fetcher ASCUEMetricsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.askue.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Parse organization ID from URL
		idStr := chi.URLParam(r, "id")
		orgID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		log = log.With(slog.Int64("organization_id", orgID))

		// Fetch all ASCUE metrics
		metricsMap, err := fetcher.FetchAll(r.Context())
		if err != nil {
			log.Error("failed to fetch ASCUE metrics", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to fetch ASCUE metrics"))
			return
		}

		// Find metrics for the requested organization
		metrics, found := metricsMap[orgID]
		if !found {
			log.Warn("organization not found in ASCUE configuration")
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.NotFound("Organization not found in ASCUE configuration"))
			return
		}

		log.Info("successfully retrieved ASCUE metrics")
		render.JSON(w, r, metrics)
	}
}
