package add

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
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// Request (JSON DTO)
type addRequest struct {
	Name                 string    `json:"name" validate:"required"`
	Description          *string   `json:"description,omitempty"`
	Location             *string   `json:"location,omitempty"`
	EventDate            time.Time `json:"event_date" validate:"required"`
	ResponsibleContactID *int64    `json:"responsible_contact_id,omitempty"`
	EventTypeID          int       `json:"event_type_id" validate:"required"`
	OrganizationID       *int64    `json:"organization_id,omitempty"`

	// Fields for creating a new contact if not using existing ID
	ResponsibleFIO   *string `json:"responsible_fio,omitempty"`
	ResponsiblePhone *string `json:"responsible_phone,omitempty"`

	FileIDs []int64 `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type eventAdder interface {
	AddEvent(ctx context.Context, req dto.AddEventRequest) (int64, error)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)
}

func New(log *slog.Logger, adder eventAdder, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
		var fileIDs []int64
		var uploadResult *fileupload.UploadResult

		// Check content type and parse accordingly
		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			// Parse request from multipart form
			req, uploadResult, err = parseMultipartAddRequest(r, log, uploader, saver, categoryGetter)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			// Combine uploaded files + existing file IDs
			existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
			fileIDs = append(existingFileIDs, uploadResult.FileIDs...)

		} else {
			log.Info("processing application/json request")

			// Parse JSON (current behavior)
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			fileIDs = req.FileIDs
		}

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))

			// Cleanup uploaded files if validation fails
			if uploadResult != nil {
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Handle responsible contact - either use existing ID or create new
		var responsibleContactID int64

		if req.ResponsibleContactID != nil {
			// Use existing contact
			contactID := *req.ResponsibleContactID

			// Verify contact exists
			_, err = adder.GetContactByID(r.Context(), contactID)
			if err != nil {
				// Cleanup uploaded files if validation fails
				if uploadResult != nil {
					fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
				}

				if errors.Is(err, storage.ErrNotFound) {
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Contact with this ID does not exist"))
					return
				}
				log.Error("failed to verify contact", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to verify contact"))
				return
			}

			responsibleContactID = contactID
		} else {
			// Create new contact from name and phone
			if req.ResponsibleFIO == nil || req.ResponsiblePhone == nil {
				// Cleanup uploaded files if validation fails
				if uploadResult != nil {
					fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
				}

				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Either 'responsible_contact_id' or both 'responsible_fio' and 'responsible_phone' are required"))
				return
			}

			// Create contact
			contactReq := dto.AddContactRequest{
				Name:  *req.ResponsibleFIO,
				Phone: req.ResponsiblePhone,
			}

			contactID, err := adder.AddContact(r.Context(), contactReq)
			if err != nil {
				// Cleanup uploaded files if contact creation fails
				if uploadResult != nil {
					fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
				}

				if errors.Is(err, storage.ErrDuplicate) {
					log.Warn("duplicate contact phone", "phone", *req.ResponsiblePhone)
					render.Status(r, http.StatusConflict)
					render.JSON(w, r, resp.BadRequest("Contact with this phone already exists"))
					return
				}
				log.Error("failed to create contact", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to create contact"))
				return
			}

			responsibleContactID = contactID
			log.Info("created new contact", slog.Int64("contact_id", contactID))
		}

		// Get default "Active" status (ID = 3)
		eventStatusID := 3 // Active status

		// Create event
		eventReq := dto.AddEventRequest{
			Name:                 req.Name,
			Description:          req.Description,
			Location:             req.Location,
			EventDate:            req.EventDate,
			ResponsibleContactID: responsibleContactID,
			EventStatusID:        eventStatusID,
			EventTypeID:          req.EventTypeID,
			OrganizationID:       req.OrganizationID,
			CreatedByID:          userID,
			FileIDs:              fileIDs,
		}

		id, err := adder.AddEvent(r.Context(), eventReq)
		if err != nil {
			// Cleanup uploaded files if event creation fails
			if uploadResult != nil {
				log.Warn("event creation failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "event_type_id", req.EventTypeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid event_type_id, organization_id, or contact_id"))
				return
			}
			log.Error("failed to add event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add event"))
			return
		}

		uploadedFilesCount := 0
		if uploadResult != nil {
			uploadedFilesCount = len(uploadResult.FileIDs)
		}
		log.Info("event added successfully",
			slog.Int64("id", id),
			slog.Int("total_files", len(fileIDs)),
			slog.Int("uploaded_files", uploadedFilesCount),
		)

		render.Status(r, http.StatusCreated)
		response := addResponse{
			Response: resp.Created(),
			ID:       id,
		}
		if uploadResult != nil && len(uploadResult.UploadedFiles) > 0 {
			response.UploadedFiles = uploadResult.UploadedFiles
		}
		render.JSON(w, r, response)
	}
}

// parseMultipartAddRequest parses event data from multipart form and handles file uploads
func parseMultipartAddRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (addRequest, *fileupload.UploadResult, error) {
	const op = "events.parseMultipartAddRequest"

	// Parse name (required)
	name, err := formparser.GetFormStringRequired(r, "name")
	if err != nil {
		return addRequest{}, nil, err
	}

	// Parse event_date (required)
	eventDate, err := formparser.GetFormTimeRequired(r, "event_date", time.RFC3339)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid or missing event_date (use RFC3339 format): %w", err)
	}

	// Parse event_type_id (required)
	eventTypeIDStr := r.FormValue("event_type_id")
	if eventTypeIDStr == "" {
		return addRequest{}, nil, fmt.Errorf("event_type_id is required")
	}
	eventTypeID, err := strconv.Atoi(eventTypeIDStr)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid event_type_id: %w", err)
	}

	// Parse optional fields
	description := formparser.GetFormString(r, "description")
	location := formparser.GetFormString(r, "location")
	orgID, err := formparser.GetFormInt64(r, "organization_id")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	// Parse responsible contact - either existing ID or new contact fields
	responsibleContactID, err := formparser.GetFormInt64(r, "responsible_contact_id")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid responsible_contact_id: %w", err)
	}

	responsibleFIO := formparser.GetFormString(r, "responsible_fio")
	responsiblePhone := formparser.GetFormString(r, "responsible_phone")

	// Create request object
	req := addRequest{
		Name:                 name,
		Description:          description,
		Location:             location,
		EventDate:            eventDate,
		ResponsibleContactID: responsibleContactID,
		EventTypeID:          eventTypeID,
		OrganizationID:       orgID,
		ResponsibleFIO:       responsibleFIO,
		ResponsiblePhone:     responsiblePhone,
	}

	// Process file uploads
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		categoryGetter,
		"events",      // category name for MinIO path
		"Мероприятия", // category display name
		eventDate,
	)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
