package get

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

type OrganizationGetter interface {
	GetAllOrganizations(ctx context.Context) ([]organization.Model, error)
}

func New(log *slog.Logger, getter OrganizationGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organizations.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgs, err := getter.GetAllOrganizations(r.Context())
		if err != nil {
			log.Error("failed to get all organizations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organizations"))
			return
		}

		log.Info("successfully retrieved organizations", slog.Int("count", len(orgs)))
		render.JSON(w, r, orgs)
	}
}
