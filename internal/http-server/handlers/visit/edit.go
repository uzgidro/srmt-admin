package visit

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
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	OrganizationID  *int64  `json:"organization_id,omitempty"`
	VisitDate       *string `json:"visit_date,omitempty"`
	Description     *string `json:"description,omitempty"`
	ResponsibleName *string `json:"responsible_name,omitempty"`
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type visitEditor interface {
	EditVisit(ctx context.Context, id int64, req dto.EditVisitRequest) error
	UnlinkVisitFiles(ctx context.Context, visitID int64) error
	LinkVisitFiles(ctx context.Context, visitID int64, fileIDs []int64) error
}

func Edit(log *slog.Logger, editor visitEditor, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.visit.edit.New"
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
		var fileIDs []int64
		var shouldUpdateFiles bool
		var uploadResult *fileupload.UploadResult

		// Check content type and parse accordingly
		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			// Parse request from multipart form
			req, uploadResult, err = parseMultipartEditRequest(r, log, uploader, saver)
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

		storageReq := dto.EditVisitRequest{
			OrganizationID:  req.OrganizationID,
			Description:     req.Description,
			ResponsibleName: req.ResponsibleName,
		}

		// Parse visit_date if provided
		if req.VisitDate != nil {
			visitDate, err := time.Parse(time.RFC3339, *req.VisitDate)
			if err != nil {
				// Cleanup uploaded files if validation fails
				if uploadResult != nil {
					fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
				}

				log.Warn("invalid visit_date format", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'visit_date' format, use ISO 8601 (e.g., 2024-01-15T10:30:00Z)"))
				return
			}
			storageReq.VisitDate = &visitDate
		}

		err = editor.EditVisit(r.Context(), id, storageReq)
		if err != nil {
			// Cleanup uploaded files if update fails
			if uploadResult != nil {
				log.Warn("visit update failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("visit not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Visit not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (org_id not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to update visit", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update visit"))
			return
		}

		// Update file links if explicitly requested
		if shouldUpdateFiles {
			// Remove old links
			if err := editor.UnlinkVisitFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}

			// Add new links (if any)
			if len(fileIDs) > 0 {
				if err := editor.LinkVisitFiles(r.Context(), id, fileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("visit updated successfully",
			slog.Int64("id", id),
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

// parseMultipartEditRequest parses visit data from multipart form and handles file uploads
func parseMultipartEditRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
) (editRequest, *fileupload.UploadResult, error) {
	const op = "visit.parseMultipartEditRequest"

	// Parse optional fields
	orgID, err := formparser.GetFormInt64(r, "organization_id")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	visitDate, err := formparser.GetFormTime(r, "visit_date", time.RFC3339)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid visit_date format (use RFC3339): %w", err)
	}

	description := formparser.GetFormString(r, "description")
	responsibleName := formparser.GetFormString(r, "responsible_name")

	// Create request object
	req := editRequest{
		OrganizationID:  orgID,
		Description:     description,
		ResponsibleName: responsibleName,
	}

	// Convert visitDate to string pointer if provided
	if visitDate != nil {
		dateStr := visitDate.Format(time.RFC3339)
		req.VisitDate = &dateStr
	}

	// Process file uploads (use current time as upload date for edits)
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		"visit",
		time.Now(), // For edits, use current time
	)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart edit form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
