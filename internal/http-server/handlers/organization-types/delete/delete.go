package delete

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
)

type OrganizationTypeDeleter interface {
	DeleteOrganizationType(ctx context.Context, id string) error
}

func New(log *slog.Logger, deleter OrganizationTypeDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organization-types.delete.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id := chi.URLParam(r, "id")
		if id == "" {
			log.Warn("id is empty")
			render.JSON(w, r, resp.BadRequest("invalid request"))
			return
		}

		err := deleter.DeleteOrganizationType(r.Context(), id)
		if err != nil {
			log.Error("failed to delete organization type", sl.Err(err))
			render.JSON(w, r, resp.InternalServerError("failed to delete organization type"))
			return
		}

		log.Info("organization type deleted", slog.String("id", id))

		render.JSON(w, r, resp.OK())
	}
}
