package get

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// OrganizationGetter defines the interface for getting organization by ID
type OrganizationGetter interface {
	GetOrganizationByID(ctx context.Context, id int64) (*organization.Model, error)
}

// New creates a handler for GET /ges/{id}
func New(log *slog.Logger, getter OrganizationGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		log = log.With(slog.Int64("organization_id", id))

		org, err := getter.GetOrganizationByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("organization not found")
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Organization not found"))
				return
			}

			log.Error("failed to get organization", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organization"))
			return
		}

		log.Info("successfully retrieved organization")
		render.JSON(w, r, org)
	}
}
