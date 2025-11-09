package incidents_handler

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

type editRequest struct {
	OrganizationID *int64     `json:"organization_id,omitempty"`
	IncidentTime   *time.Time `json:"incident_time,omitempty"`
	Description    *string    `json:"description,omitempty"`
}

type incidentEditor interface {
	EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest) error
}

func Edit(log *slog.Logger, editor incidentEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.incident.update.New"
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

		storageReq := dto.EditIncidentRequest{
			OrganizationID: req.OrganizationID,
			IncidentTime:   req.IncidentTime,
			Description:    req.Description,
		}

		err = editor.EditIncident(r.Context(), id, storageReq)
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
			log.Error("failed to update incident", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update incident"))
			return
		}

		log.Info("incident updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
