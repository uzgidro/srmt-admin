package shutdowns

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"time"
)

type addRequest struct {
	OrganizationID      int64      `json:"organization_id" validate:"required"`
	StartTime           time.Time  `json:"start_time" validate:"required"`
	EndTime             *time.Time `json:"end_time,omitempty"`
	Reason              *string    `json:"reason,omitempty"`
	GenerationLossMwh   *float64   `json:"generation_loss,omitempty"`
	ReportedByContactID *int64     `json:"reported_by_contact_id,omitempty"`

	IdleDischargeVolume *float64 `json:"idle_discharge_volume,omitempty"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type ShutdownAdder interface {
	AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error)
}

func Add(log *slog.Logger, adder ShutdownAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
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

		if req.IdleDischargeVolume != nil && req.EndTime == nil {
			log.Warn("validation failed: end_time is required if idle_discharge_volume is provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_time is required when providing idle_discharge_volume"))
			return
		}
		if req.EndTime != nil && !req.EndTime.After(req.StartTime) {
			log.Warn("validation failed: end_time must be after start_time")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_time must be after start_time"))
			return
		}

		storageReq := dto.AddShutdownRequest{
			OrganizationID:      req.OrganizationID,
			StartTime:           req.StartTime,
			EndTime:             req.EndTime,
			Reason:              req.Reason,
			GenerationLossMwh:   req.GenerationLossMwh,
			ReportedByContactID: req.ReportedByContactID,
			CreatedByUserID:     userID,

			IdleDischargeVolumeThousandM3: req.IdleDischargeVolume,
		}

		id, err := adder.AddShutdown(r.Context(), storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation (org or contact not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id or reported_by_contact_id"))
				return
			}
			log.Error("failed to add shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add shutdown"))
			return
		}

		log.Info("shutdown added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{Response: resp.Created(), ID: id})
	}
}
