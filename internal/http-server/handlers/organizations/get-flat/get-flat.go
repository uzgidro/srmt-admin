package get_flat

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/organization"
)

type OrganizationFlatGetter interface {
	GetFlatOrganizations(ctx context.Context, orgType *string) ([]*organization.Model, error)
}

func New(log *slog.Logger, getter OrganizationFlatGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organizations.get_flat.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgTypeQuery := r.URL.Query().Get("type")
		var orgType *string
		if orgTypeQuery != "" {
			orgType = &orgTypeQuery
			log = log.With(slog.String("type", *orgType))
		}

		orgs, err := getter.GetFlatOrganizations(r.Context(), orgType)
		if err != nil {
			log.Error("failed to get flat organizations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organizations"))
			return
		}

		log.Info("successfully retrieved flat organizations", slog.Int("count", len(orgs)))
		render.JSON(w, r, orgs)
	}
}
