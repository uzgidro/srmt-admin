package shutdowns

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"
	"time"
)

// Request (JSON DTO)
type editRequest struct {
	OrganizationID      *int64      `json:"organization_id,omitempty"`
	StartTime           *time.Time  `json:"start_time,omitempty"`
	EndTime             **time.Time `json:"end_time,omitempty"`
	Reason              *string     `json:"reason,omitempty"`
	GenerationLossMwh   *float64    `json:"generation_loss,omitempty"`
	ReportedByContactID *int64      `json:"reported_by_contact_id,omitempty"`

	IdleDischargeVolume *float64 `json:"idle_discharge_volume,omitempty"`
}

type shutdownEditor interface {
	EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error
}

func Edit(log *slog.Logger, editor shutdownEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if req.IdleDischargeVolume != nil && *req.IdleDischargeVolume > 0 &&
			(req.EndTime == nil || (req.EndTime != nil && *req.EndTime == nil)) {
			log.Warn("validation failed: end_time is required if idle_discharge_volume is provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_time is required when providing idle_discharge_volume"))
			return
		}

		storageReq := dto.EditShutdownRequest{
			OrganizationID:      req.OrganizationID,
			StartTime:           req.StartTime,
			EndTime:             req.EndTime,
			Reason:              req.Reason,
			GenerationLossMwh:   req.GenerationLossMwh,
			ReportedByContactID: req.ReportedByContactID,

			IdleDischargeVolumeThousandM3: req.IdleDischargeVolume,
		}

		err = editor.EditShutdown(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("incident not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Incident not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (org_id not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to update shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update shutdown"))
			return
		}

		log.Info("shutdown updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
