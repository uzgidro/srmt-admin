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

type fastCallDeleter interface {
	DeleteFastCall(ctx context.Context, id int64) error
}

func New(log *slog.Logger, deleter fastCallDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.fast_call.delete.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get fast call ID from URL parameter
		idStr := chi.URLParam(r, "id")
		fastCallID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("invalid fast call ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid fast call ID"))
			return
		}

		// Delete fast call
		err = deleter.DeleteFastCall(r.Context(), fastCallID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Fast call not found"))
				return
			}

			log.Error("failed to delete fast call", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete fast call"))
			return
		}

		log.Info("fast call deleted successfully", slog.Int64("fast_call_id", fastCallID))
		render.JSON(w, r, resp.OK())
	}
}
