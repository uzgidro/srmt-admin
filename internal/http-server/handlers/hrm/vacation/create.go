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

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type VacationCreator interface {
	Create(ctx context.Context, req dto.CreateVacationRequest, createdBy int64) (int64, error)
}

type CreateResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

func Create(log *slog.Logger, svc VacationCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Create"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		var req dto.CreateVacationRequest
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

		id, err := svc.Create(r.Context(), req, claims.ContactID)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrStartDateInPast):
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Start date cannot be in the past"))
			case errors.Is(err, storage.ErrInvalidDateRange):
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid date range"))
			case errors.Is(err, storage.ErrInsufficientBalance):
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Insufficient vacation balance"))
			case errors.Is(err, storage.ErrVacationOverlap):
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Vacation dates overlap with an existing vacation"))
			case errors.Is(err, storage.ErrBlockedPeriod):
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Vacation dates fall within a department blocked period"))
			case errors.Is(err, storage.ErrForeignKeyViolation):
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee or substitute ID"))
			default:
				log.Error("failed to create vacation", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to create vacation"))
			}
			return
		}

		log.Info("vacation created", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, CreateResponse{Response: resp.OK(), ID: id})
	}
}
