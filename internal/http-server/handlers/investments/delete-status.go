package investments

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

type investmentStatusDeleter interface {
	DeleteInvestmentStatus(ctx context.Context, id int) error
}

func DeleteStatus(log *slog.Logger, deleter investmentStatusDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.delete-status"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = deleter.DeleteInvestmentStatus(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("investment status not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Investment status not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("cannot delete investment status - it is being used", slog.Int("id", id))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Cannot delete investment status - it is being used by investments"))
				return
			}
			log.Error("failed to delete investment status", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete investment status"))
			return
		}

		log.Info("investment status deleted successfully", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}
