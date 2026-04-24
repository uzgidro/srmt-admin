package gesreport

import (
	"context"
	"log/slog"
	"net/http"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// FrozenDefaultLister is the local dependency contract for GET.
type FrozenDefaultLister interface {
	ListFrozenDefaults(ctx context.Context) ([]model.FrozenDefault, error)
}

// ListFrozenDefaults returns all frozen-default entries visible to the caller.
//
// Visibility mirrors filterGESConfigsForCaller exactly: sc/rais see everything;
// everyone else sees entries whose organization_id matches their own claim OR
// whose parent organization (cascade) does. The repo populates CascadeID via
// JOIN on organizations.parent_organization_id so a cascade-role user sees the
// freezes for every station in their cascade — matching their write access
// granted by CheckCascadeStationAccess in PUT/DELETE.
func ListFrozenDefaults(log *slog.Logger, repo FrozenDefaultLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.ListFrozenDefaults"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		entries, err := repo.ListFrozenDefaults(r.Context())
		if err != nil {
			log.Error("failed to list frozen defaults", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve frozen defaults"))
			return
		}

		entries = filterFrozenDefaultsForCaller(r.Context(), entries)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, entries)
	}
}

// filterFrozenDefaultsForCaller restricts the list to what the current user
// may see. sc/rais get the unfiltered slice. Other roles get entries for
// their own organization and entries belonging to stations in their cascade
// (CascadeID == claims.OrganizationID). When claims are missing or the org
// id is zero we deny by returning an empty slice — defence-in-depth in case
// the route group middleware is ever bypassed.
func filterFrozenDefaultsForCaller(ctx context.Context, entries []model.FrozenDefault) []model.FrozenDefault {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return []model.FrozenDefault{}
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return entries
		}
	}
	if claims.OrganizationID == 0 {
		return []model.FrozenDefault{}
	}
	filtered := make([]model.FrozenDefault, 0, len(entries))
	for _, e := range entries {
		if e.OrganizationID == claims.OrganizationID {
			filtered = append(filtered, e)
			continue
		}
		if e.CascadeID != nil && *e.CascadeID == claims.OrganizationID {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
