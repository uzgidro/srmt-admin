package edit

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
	"srmt-admin/internal/lib/model/event"
	"srmt-admin/internal/storage"
	"strconv"
	"time"
)

// Request - JSON DTO for editing an event
type Request struct {
	Name                 *string    `json:"name,omitempty"`
	Description          *string    `json:"description,omitempty"`
	Location             *string    `json:"location,omitempty"`
	EventDate            *time.Time `json:"event_date,omitempty"`
	ResponsibleContactID *int64     `json:"responsible_contact_id,omitempty"`
	EventStatusID        *int       `json:"event_status_id,omitempty"`
	EventTypeID          *int       `json:"event_type_id,omitempty"`
	OrganizationID       *int64     `json:"organization_id,omitempty"`
	FileIDs              []int64    `json:"file_ids,omitempty"` // Replaces all existing file links
}

// EventEditor defines repository interface for event updates
type EventEditor interface {
	EditEvent(ctx context.Context, eventID int64, req dto.EditEventRequest) error
	GetEventByID(ctx context.Context, id int64) (*event.Model, error)
}

func New(log *slog.Logger, editor EventEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.edit.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Get event ID from URL
		idStr := chi.URLParam(r, "id")
		eventID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("invalid event ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid event ID"))
			return
		}

		// 2. Verify event exists
		_, err = editor.GetEventByID(r.Context(), eventID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Event not found"))
				return
			}
			log.Error("failed to get event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to verify event"))
			return
		}

		// 3. Decode request body
		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// 4. Get user ID from context
		userID := int64(1) // TODO: Get from auth context
		// userID := r.Context().Value("user_id").(int64)

		// 5. Build storage request
		storageReq := dto.EditEventRequest{
			Name:                 req.Name,
			Description:          req.Description,
			Location:             req.Location,
			EventDate:            req.EventDate,
			ResponsibleContactID: req.ResponsibleContactID,
			EventStatusID:        req.EventStatusID,
			EventTypeID:          req.EventTypeID,
			OrganizationID:       req.OrganizationID,
			UpdatedByID:          userID,
			FileIDs:              req.FileIDs,
		}

		// 6. Update event
		err = editor.EditEvent(r.Context(), eventID, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Event not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation during update")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid event_type_id, event_status_id, organization_id, or contact_id"))
				return
			}

			log.Error("failed to update event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update event"))
			return
		}

		log.Info("event updated successfully", slog.Int64("event_id", eventID))
		render.JSON(w, r, resp.OK())
	}
}
