package vacation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Approver interface {
	Approve(ctx context.Context, id int64, approvedBy int64) error
}

func Approve(log *slog.Logger, svc Approver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Approve"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		if err := svc.Approve(r.Context(), id, claims.ContactID); err != nil {
			if errors.Is(err, storage.ErrVacationNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			if errors.Is(err, storage.ErrInvalidStatus) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Only pending vacations can be approved"))
				return
			}
			log.Error("failed to approve vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to approve vacation"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
