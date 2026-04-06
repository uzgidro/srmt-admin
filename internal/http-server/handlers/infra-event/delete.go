package infraevent

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type eventDeleter interface {
	DeleteInfraEvent(ctx context.Context, id int64) error
}

func Delete(log *slog.Logger, deleter eventDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event.delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = deleter.DeleteInfraEvent(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Event not found"))
				return
			}
			log.Error("failed to delete infra event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete event"))
			return
		}

		log.Info("infra event deleted", slog.Int64("id", id))
		w.WriteHeader(http.StatusNoContent)
	}
}
