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

type PositionDeleter interface {
	DeletePosition(ctx context.Context, id int64) error
}

func New(log *slog.Logger, deleter PositionDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.position.delete.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter, must be a number"))
			return
		}

		err = deleter.DeletePosition(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("position not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Position not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("position has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete position: it is in use by contacts"))
				return
			}
			log.Error("failed to delete position", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete position"))
			return
		}

		log.Info("position deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
