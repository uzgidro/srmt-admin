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

type DischargeDeleter interface {
	DeleteDischarge(ctx context.Context, id int64) error
}

func New(log *slog.Logger, deleter DischargeDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.delete.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		dischargeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid discharge ID format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid discharge ID"))
			return
		}

		err = deleter.DeleteDischarge(r.Context(), dischargeID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("discharge not found", "id", dischargeID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Discharge not found"))
				return
			}
			log.Error("failed to delete discharge", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete discharge"))
			return
		}

		log.Info("discharge deleted successfully", slog.Int64("id", dischargeID))
		render.Status(r, http.StatusNoContent)
	}
}
