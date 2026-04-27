package reservoirflood

import (
	"context"
	"log/slog"
	"net/http"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/reservoir-flood"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ConfigGetter interface {
	GetAllReservoirFloodConfigs(ctx context.Context) ([]model.Config, error)
}

func GetConfigs(log *slog.Logger, repo ConfigGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoir-flood.GetConfigs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		configs, err := repo.GetAllReservoirFloodConfigs(r.Context())
		if err != nil {
			log.Error("failed to get configs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve configs"))
			return
		}

		// Filter for reservoir_duty: only their own org.
		configs = filterConfigsForCaller(r.Context(), configs)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, configs)
	}
}

func filterConfigsForCaller(ctx context.Context, list []model.Config) []model.Config {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return []model.Config{}
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return list
		}
	}
	if claims.OrganizationID == 0 {
		return []model.Config{}
	}
	out := make([]model.Config, 0, len(list))
	for _, c := range list {
		if c.OrganizationID == claims.OrganizationID {
			out = append(out, c)
		}
	}
	return out
}
