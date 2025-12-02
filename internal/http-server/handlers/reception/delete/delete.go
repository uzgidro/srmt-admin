package delete

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

type receptionDeleter interface {
	DeleteReception(ctx context.Context, id int64) error
}

func New(log *slog.Logger, deleter receptionDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reception.delete.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get reception ID from URL parameter
		idStr := chi.URLParam(r, "id")
		receptionID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("invalid reception ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid reception ID"))
			return
		}

		// Delete reception
		err = deleter.DeleteReception(r.Context(), receptionID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Reception not found"))
				return
			}

			log.Error("failed to delete reception", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete reception"))
			return
		}

		log.Info("reception deleted successfully", slog.Int64("reception_id", receptionID))
		render.JSON(w, r, resp.OK())
	}
}
