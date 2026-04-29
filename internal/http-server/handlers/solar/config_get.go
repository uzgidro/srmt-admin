package solar

import (
	"context"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/solar"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ConfigGetter interface {
	GetAllSolarConfigs(ctx context.Context) ([]model.Config, error)
}

func GetConfigs(log *slog.Logger, repo ConfigGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.solar.GetConfigs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// No special filtering — sc/rais/cascade all see the full config list.
		// (Solar configs are organization-level metadata; visibility is shared.)
		configs, err := repo.GetAllSolarConfigs(r.Context())
		if err != nil {
			log.Error("failed to get solar configs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve configs"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, configs)
	}
}
