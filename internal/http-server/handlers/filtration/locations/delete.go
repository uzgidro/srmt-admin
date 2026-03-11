package locations

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type LocationDeleter interface {
	DeleteFiltrationLocation(ctx context.Context, id int64) error
}

func Delete(log *slog.Logger, deleter LocationDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.locations.Delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		if err := deleter.DeleteFiltrationLocation(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("location not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Location not found"))
				return
			}
			log.Error("failed to delete location", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete location"))
			return
		}

		log.Info("location deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}
