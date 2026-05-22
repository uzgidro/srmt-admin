package reservoirflood

import (
	"context"
	"log/slog"
	"net/http"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/reservoir-flood"
	"srmt-admin/internal/lib/service/auth"

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

		// Reject broken-account state (non-admin without org) BEFORE the repo,
		// symmetric with the upsert path. Returning 200 with [] would silently
		// mask a misconfigured user.
		if !callerIsAdmin(r.Context()) {
			claims, ok := mwauth.ClaimsFromContext(r.Context())
			if !ok || claims == nil || len(claims.OrganizationIDs) == 0 {
				log.Warn("non-admin caller without organization id")
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("user has no organization assigned"))
				return
			}
		}

		configs, err := repo.GetAllReservoirFloodConfigs(r.Context())
		if err != nil {
			log.Error("failed to get configs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve configs"))
			return
		}

		// Filter for reservoir_flood: only their own org.
		configs = filterConfigsForCaller(r.Context(), configs)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, configs)
	}
}

// filterConfigsForCaller restricts the response to configs the caller is
// allowed to see. sc/rais see everything. Other roles (typically
// reservoir_flood) see only configs for orgs in their assigned org set. The
// handler MUST have already enforced a non-empty claims.OrganizationIDs for
// non-admins — see GetConfigs.
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
	out := make([]model.Config, 0, len(list))
	for _, c := range list {
		if auth.ContainsOrg(claims.OrganizationIDs, c.OrganizationID) {
			out = append(out, c)
		}
	}
	return out
}
