package delete

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"
)

type OrganizationDeleter interface {
	DeleteOrganization(ctx context.Context, id int64) error
}

func New(log *slog.Logger, deleter OrganizationDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organizations.delete.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid organization ID format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid organization ID"))
			return
		}

		err = deleter.DeleteOrganization(r.Context(), orgID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("organization not found, nothing to delete", "id", orgID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Organization not found"))
				return
			}
			log.Error("failed to delete organization", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete organization"))
			return
		}

		log.Info("organization deleted successfully", slog.Int64("id", orgID))
		render.Status(r, http.StatusNoContent)
	}
}
