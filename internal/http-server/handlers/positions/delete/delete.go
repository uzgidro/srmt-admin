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
		const op = "handlers.positions.delete.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		positionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid position ID format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid position ID"))
			return
		}

		err = deleter.DeletePosition(r.Context(), positionID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("position not found, nothing to delete", "id", positionID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Position not found"))
				return
			}
			log.Error("failed to delete position", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete position"))
			return
		}

		log.Info("position deleted successfully", slog.Int64("id", positionID))
		render.Status(r, http.StatusNoContent)
	}
}
