package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// FileUploader defines interface for file storage operations
type FileUploader interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	DeleteFile(ctx context.Context, objectName string) error
}

// EventAdder defines repository interface for event creation
type EventAdder interface {
	AddEvent(ctx context.Context, req dto.AddEventRequest) (int64, error)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)
	AddFile(ctx context.Context, fileData file.Model) (int64, error)
	GetEventsCategory(ctx context.Context) (category.Model, error)
}

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

// New creates a new HTTP handler for adding events with file uploads
func New(log *slog.Logger, uploader FileUploader, adder EventAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.event.add.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Parse multipart form (max 100MB for multiple files)
		const maxUploadSize = 100 * 1024 * 1024
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			log.Error("failed to parse multipart form", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request or files too large"))
			return
		}

		// 2. Parse required fields
		name := r.FormValue("name")
		if name == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Field 'name' is required"))
			return
		}

		eventDateStr := r.FormValue("event_date")
		if eventDateStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Field 'event_date' is required"))
			return
		}

		eventDate, err := time.Parse(time.RFC3339, eventDateStr)
		if err != nil {
			log.Error("invalid event_date format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'event_date' format. Use ISO 8601 (RFC3339)"))
			return
		}

		eventTypeIDStr := r.FormValue("event_type_id")
		eventTypeID, err := strconv.Atoi(eventTypeIDStr)
		if err != nil {
			log.Error("invalid event_type_id", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid or missing 'event_type_id'"))
			return
		}

		// 3. Parse optional fields
		description := r.FormValue("description")
		var descPtr *string
		if description != "" {
			descPtr = &description
		}

		location := r.FormValue("location")
		var locPtr *string
		if location != "" {
			locPtr = &location
		}

		var orgIDPtr *int64
		if orgIDStr := r.FormValue("organization_id"); orgIDStr != "" {
			orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Error("invalid organization_id", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id'"))
				return
			}
			orgIDPtr = &orgID
		}

		// 4. Get user ID from context (assuming auth middleware sets this)
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		// 5. Handle responsible contact - either use existing ID or create new
		var responsibleContactID int64

		if contactIDStr := r.FormValue("responsible_contact_id"); contactIDStr != "" {
			// Use existing contact
			contactID, err := strconv.ParseInt(contactIDStr, 10, 64)
			if err != nil {
				log.Error("invalid responsible_contact_id", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'responsible_contact_id'"))
				return
			}

			// Verify contact exists
			_, err = adder.GetContactByID(r.Context(), contactID)
			if err != nil {
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
			// Create new contact from FIO and phone
			fio := r.FormValue("responsible_fio")
			phone := r.FormValue("responsible_phone")

			if fio == "" || phone == "" {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Either 'responsible_contact_id' or both 'responsible_fio' and 'responsible_phone' are required"))
				return
			}

			// Create contact
			contactReq := dto.AddContactRequest{
				FIO:   fio,
				Phone: &phone,
			}

			contactID, err := adder.AddContact(r.Context(), contactReq)
			if err != nil {
				if errors.Is(err, storage.ErrDuplicate) {
					log.Warn("duplicate contact phone", "phone", phone)
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

		// 6. Get default "Active" status (ID = 3 based on migration)
		eventStatusID := 3 // Active status

		// 7. Handle file uploads
		var uploadedFiles []file.Model
		var fileIDs []int64

		files := r.MultipartForm.File["files"]
		if len(files) > 0 {
			cat, err := adder.GetEventsCategory(r.Context())
			if err != nil {
				log.Error("failed to get file category", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("File category not configured"))
				return
			}

			for _, fileHeader := range files {
				fileReader, err := fileHeader.Open()
				if err != nil {
					log.Error("failed to open uploaded file", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Failed to read uploaded file"))
					return
				}
				defer fileReader.Close()

				// Generate unique object key
				datePrefix := time.Now().Format("2006/01/02")
				objectKey := fmt.Sprintf("%s/%s/%s%s",
					cat.DisplayName,
					datePrefix,
					uuid.New().String(),
					filepath.Ext(fileHeader.Filename),
				)

				// Upload to MinIO
				err = uploader.UploadFile(r.Context(), objectKey, fileReader, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
				if err != nil {
					log.Error("failed to upload file to storage", sl.Err(err))
					// Clean up previously uploaded files
					for _, uf := range uploadedFiles {
						_ = uploader.DeleteFile(r.Context(), uf.ObjectKey)
					}
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to upload file"))
					return
				}

				// Save file metadata
				fileMeta := file.Model{
					FileName:   fileHeader.Filename,
					ObjectKey:  objectKey,
					CategoryID: cat.ID,
					MimeType:   fileHeader.Header.Get("Content-Type"),
					SizeBytes:  fileHeader.Size,
					CreatedAt:  time.Now(),
				}

				fileID, err := adder.AddFile(r.Context(), fileMeta)
				if err != nil {
					log.Error("failed to save file metadata", sl.Err(err))
					// Clean up uploaded files
					_ = uploader.DeleteFile(r.Context(), objectKey)
					for _, uf := range uploadedFiles {
						_ = uploader.DeleteFile(r.Context(), uf.ObjectKey)
					}
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to save file metadata"))
					return
				}

				fileMeta.ID = fileID
				uploadedFiles = append(uploadedFiles, fileMeta)
				fileIDs = append(fileIDs, fileID)

				log.Info("file uploaded successfully", slog.Int64("file_id", fileID), slog.String("object_key", objectKey))
			}
		}

		// 8. Create event
		eventReq := dto.AddEventRequest{
			Name:                 name,
			Description:          descPtr,
			Location:             locPtr,
			EventDate:            eventDate,
			ResponsibleContactID: responsibleContactID,
			EventStatusID:        eventStatusID,
			EventTypeID:          eventTypeID,
			OrganizationID:       orgIDPtr,
			CreatedByID:          userID,
			FileIDs:              fileIDs,
		}

		eventID, err := adder.AddEvent(r.Context(), eventReq)
		if err != nil {
			// Clean up uploaded files on event creation failure
			for _, uf := range uploadedFiles {
				if delErr := uploader.DeleteFile(r.Context(), uf.ObjectKey); delErr != nil {
					log.Error("failed to cleanup file after event creation failure",
						sl.Err(delErr),
						slog.String("object_key", uf.ObjectKey),
					)
				}
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "event_type_id", eventTypeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid event_type_id, organization_id, or contact_id"))
				return
			}

			log.Error("failed to create event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create event"))
			return
		}

		log.Info("event created successfully",
			slog.Int64("event_id", eventID),
			slog.Int("files_count", len(fileIDs)),
		)

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: eventID})
	}
}
