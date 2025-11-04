package add

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"time"
)

type Request struct {
	OrganizationID int64      `json:"organization_id" validate:"required"`
	StartedAt      time.Time  `json:"started_at" validate:"required"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	FlowRate       float64    `json:"flow_rate" validate:"required,gt=0"`
	Reason         string     `json:"reason" validate:"required"`
}

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

type DischargeAdder interface {
	AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason string) (int64, error)
}

func New(log *slog.Logger, adder DischargeAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Получаем ID пользователя из контекста
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Передаем новое поле в метод репозитория
		id, err := adder.AddDischarge(r.Context(), req.OrganizationID, userID, req.StartedAt, req.EndedAt, req.FlowRate, req.Reason)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("organization not found", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to add discharge", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add discharge"))
			return
		}

		log.Info("discharge added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: id})
	}
}
