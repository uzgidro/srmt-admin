package incidents_handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	OrganizationID *int64     `json:"organization_id,omitempty"`
	IncidentTime   *time.Time `json:"incident_time,omitempty"`
	Description    *string    `json:"description,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
}

type incidentEditor interface {
	EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest) error
	UnlinkIncidentFiles(ctx context.Context, incidentID int64) error
	LinkIncidentFiles(ctx context.Context, incidentID int64, fileIDs []int64) error
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

		// Build storage request
		storageReq := dto.EditIncidentRequest{
			OrganizationID: req.OrganizationID,
			IncidentTime:   req.IncidentTime,
			Description:    req.Description,
			FileIDs:        req.FileIDs,
		}

		// Update incident
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

		// Update file links if explicitly requested
		if req.FileIDs != nil {
			if err := editor.UnlinkIncidentFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}
			if len(req.FileIDs) > 0 {
				if err := editor.LinkIncidentFiles(r.Context(), id, req.FileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("incident updated successfully", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
