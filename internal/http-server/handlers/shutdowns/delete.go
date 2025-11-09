package shutdowns

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

type shutdownDeleter interface {
	DeleteShutdown(ctx context.Context, id int64) error
}

func Delete(log *slog.Logger, deleter shutdownDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = deleter.DeleteShutdown(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				if errors.Is(err, storage.ErrNotFound) {
					log.Warn("incident not found", slog.Int64("id", id))
					render.Status(r, http.StatusNotFound)
					render.JSON(w, r, resp.NotFound("Incident not found"))
					return
				}
			}
			log.Error("failed to delete shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete shutdown"))
			return
		}

		log.Info("shutdown deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
