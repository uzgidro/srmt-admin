package access

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

type CardUnblocker interface {
	UnblockCard(ctx context.Context, id int64) error
}

func UnblockCard(log *slog.Logger, svc CardUnblocker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.UnblockCard"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		if err := svc.UnblockCard(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrAccessCardNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access card not found"))
				return
			}
			if errors.Is(err, storage.ErrInvalidStatus) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Card is not in blocked status"))
				return
			}
			log.Error("failed to unblock access card", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to unblock access card"))
			return
		}

		render.JSON(w, r, resp.OK())
	}
}
