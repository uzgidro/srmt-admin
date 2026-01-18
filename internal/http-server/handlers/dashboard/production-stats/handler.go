package productionstats

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
	GetGesProductionStats(ctx context.Context) (*gesproduction.StatsResponse, error)
}

func New(log *slog.Logger, provider Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.dashboard.production-stats.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		data, err := provider.GetGesProductionStats(r.Context())
		if err != nil {
			log.Error("failed to get ges production stats", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get data"))
			return
		}

		if data == nil {
			log.Info("no ges production data found")
			render.JSON(w, r, nil)
			return
		}

		render.JSON(w, r, data)
	}
}
