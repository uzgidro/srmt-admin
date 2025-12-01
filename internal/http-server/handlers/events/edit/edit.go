package edit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/event"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// Request - JSON DTO for editing an event
type editRequest struct {
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

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

// EventEditor defines repository interface for event updates
type eventEditor interface {
	EditEvent(ctx context.Context, eventID int64, req dto.EditEventRequest) error
	GetEventByID(ctx context.Context, id int64) (*event.Model, error)
	UnlinkEventFiles(ctx context.Context, eventID int64) error
	LinkEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error
}

func New(log *slog.Logger, editor eventEditor, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
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

		// 3. Get user ID from context
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req editRequest
		var fileIDs []int64
		var shouldUpdateFiles bool
		var uploadResult *fileupload.UploadResult

		// Check content type and parse accordingly
		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			// Parse request from multipart form
			req, uploadResult, err = parseMultipartEditRequest(r, log, uploader, saver, categoryGetter)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			// Check if files field is present in form
			if formparser.HasFormField(r, "file_ids") || len(uploadResult.FileIDs) > 0 {
				shouldUpdateFiles = true
				// Get existing file IDs from form
				existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
				// Combine uploaded + existing
				fileIDs = append(existingFileIDs, uploadResult.FileIDs...)
			}

		} else {
			log.Info("processing application/json request")

			// Parse JSON (current behavior)
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			// In JSON, if file_ids is present (even empty array), update files
			// This fixes the issue where you couldn't remove all files
			if req.FileIDs != nil {
				shouldUpdateFiles = true
				fileIDs = req.FileIDs
			}
		}

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
			FileIDs:              fileIDs,
		}

		// 6. Update event
		err = editor.EditEvent(r.Context(), eventID, storageReq)
		if err != nil {
			// Cleanup uploaded files if update fails
			if uploadResult != nil {
				log.Warn("event update failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

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

		// Update file links if explicitly requested
		if shouldUpdateFiles {
			// Remove old links
			if err := editor.UnlinkEventFiles(r.Context(), eventID); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}

			// Add new links (if any)
			if len(fileIDs) > 0 {
				if err := editor.LinkEventFiles(r.Context(), eventID, fileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("event updated successfully",
			slog.Int64("event_id", eventID),
			slog.Bool("files_updated", shouldUpdateFiles),
			slog.Int("total_files", len(fileIDs)),
		)

		response := editResponse{
			Response: resp.OK(),
		}
		if uploadResult != nil && len(uploadResult.UploadedFiles) > 0 {
			response.UploadedFiles = uploadResult.UploadedFiles
		}
		render.JSON(w, r, response)
	}
}

// parseMultipartEditRequest parses event data from multipart form and handles file uploads
func parseMultipartEditRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (editRequest, *fileupload.UploadResult, error) {
	const op = "events.parseMultipartEditRequest"

	// Parse optional fields
	name := formparser.GetFormString(r, "name")
	description := formparser.GetFormString(r, "description")
	location := formparser.GetFormString(r, "location")

	eventDate, err := formparser.GetFormTime(r, "event_date", time.RFC3339)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid event_date format (use RFC3339): %w", err)
	}

	responsibleContactID, err := formparser.GetFormInt64(r, "responsible_contact_id")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid responsible_contact_id: %w", err)
	}

	// Parse event_status_id (optional)
	var eventStatusID *int
	if eventStatusIDStr := r.FormValue("event_status_id"); eventStatusIDStr != "" {
		statusID, err := strconv.Atoi(eventStatusIDStr)
		if err != nil {
			return editRequest{}, nil, fmt.Errorf("invalid event_status_id: %w", err)
		}
		eventStatusID = &statusID
	}

	// Parse event_type_id (optional)
	var eventTypeID *int
	if eventTypeIDStr := r.FormValue("event_type_id"); eventTypeIDStr != "" {
		typeID, err := strconv.Atoi(eventTypeIDStr)
		if err != nil {
			return editRequest{}, nil, fmt.Errorf("invalid event_type_id: %w", err)
		}
		eventTypeID = &typeID
	}

	orgID, err := formparser.GetFormInt64(r, "organization_id")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	// Create request object
	req := editRequest{
		Name:                 name,
		Description:          description,
		Location:             location,
		EventDate:            eventDate,
		ResponsibleContactID: responsibleContactID,
		EventStatusID:        eventStatusID,
		EventTypeID:          eventTypeID,
		OrganizationID:       orgID,
	}

	// Process file uploads (use current time as upload date for edits)
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		categoryGetter,
		"events",      // category name for MinIO path
		"Мероприятия", // category display name
		time.Now(),    // For edits, use current time
	)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart edit form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
