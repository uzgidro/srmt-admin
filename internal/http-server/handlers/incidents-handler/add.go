package incidents_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// Request (JSON DTO)
type addRequest struct {
	OrganizationID *int64    `json:"organization_id,omitempty"`
	IncidentTime   time.Time `json:"incident_time" validate:"required"`
	Description    string    `json:"description" validate:"required"`
	FileIDs        []int64   `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type incidentAdder interface {
	AddIncident(ctx context.Context, orgID *int64, incidentTime time.Time, description string, createdByID int64) (int64, error)
	LinkIncidentFiles(ctx context.Context, incidentID int64, fileIDs []int64) error
}

func Add(log *slog.Logger, adder incidentAdder, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.incident.add.New"
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

		// Create incident
		id, err := adder.AddIncident(
			r.Context(),
			req.OrganizationID,
			req.IncidentTime,
			req.Description,
			userID,
		)
		if err != nil {
			// Cleanup uploaded files if incident creation fails
			if uploadResult != nil {
				log.Warn("incident creation failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				orgIDVal := "nil"
				if req.OrganizationID != nil {
					orgIDVal = fmt.Sprintf("%d", *req.OrganizationID)
				}
				log.Warn("organization not found", "org_id", orgIDVal)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to add incident", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add incident"))
			return
		}

		// Link files if provided
		if len(fileIDs) > 0 {
			if err := adder.LinkIncidentFiles(r.Context(), id, fileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
				// Don't fail the request, just log the error
			}
		}

		uploadedFilesCount := 0
		if uploadResult != nil {
			uploadedFilesCount = len(uploadResult.FileIDs)
		}
		log.Info("incident added successfully",
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

// parseMultipartAddRequest parses incident data from multipart form and handles file uploads
func parseMultipartAddRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (addRequest, *fileupload.UploadResult, error) {
	const op = "incidents_handler.parseMultipartAddRequest"

	// Parse organization_id (optional)
	orgID, err := formparser.GetFormInt64(r, "organization_id")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	// Parse incident_time (required)
	incidentTime, err := formparser.GetFormTimeRequired(r, "incident_time", time.RFC3339)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid or missing incident_time (use RFC3339 format): %w", err)
	}

	// Parse description (required)
	description, err := formparser.GetFormStringRequired(r, "description")
	if err != nil {
		return addRequest{}, nil, err
	}

	// Create request object
	req := addRequest{
		OrganizationID: orgID,
		IncidentTime:   incidentTime,
		Description:    description,
	}

	// Process file uploads
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		categoryGetter,
		"incidents", // category name for MinIO path
		"Инциденты", // category display name
		incidentTime,
	)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
