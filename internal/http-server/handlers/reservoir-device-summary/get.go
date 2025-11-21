package reservoirdevicesummary

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	reservoirdevicesummary "srmt-admin/internal/lib/model/reservoir-device-summary"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type reservoirDeviceSummaryGetter interface {
	GetReservoirDeviceSummary(ctx context.Context) ([]*reservoirdevicesummary.ResponseModel, error)
}

func Get(log *slog.Logger, getter reservoirDeviceSummaryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoirdevicesummary.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		summaries, err := getter.GetReservoirDeviceSummary(r.Context())
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
