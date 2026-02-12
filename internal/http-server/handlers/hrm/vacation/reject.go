package vacation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Rejecter interface {
	Reject(ctx context.Context, id int64, rejectedBy int64, reason string) error
}

func Reject(log *slog.Logger, svc Rejecter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Reject"
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

		var req dto.RejectVacationRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		if err := svc.Reject(r.Context(), id, claims.ContactID, req.Reason); err != nil {
			if errors.Is(err, storage.ErrVacationNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			if errors.Is(err, storage.ErrInvalidStatus) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Only pending vacations can be rejected"))
				return
			}
			log.Error("failed to reject vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to reject vacation"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
