package get

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/organization-type"
)

type OrganizationTypeGetter interface {
	GetAllOrganizationTypes(ctx context.Context) ([]organization_type.Model, error)
}

func New(log *slog.Logger, getter OrganizationTypeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organization-types.get.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		organizationTypes, err := getter.GetAllOrganizationTypes(r.Context())
		if err != nil {
			log.Error("failed to get all organization types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organization types"))
			return
		}

		log.Info("successfully retrieved all organization types", slog.Int("count", len(organizationTypes)))

		render.JSON(w, r, organizationTypes)
	}
}
