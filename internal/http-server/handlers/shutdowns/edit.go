package shutdowns

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
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
	FileIDs             []int64  `json:"file_ids,omitempty"`
}

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type shutdownEditor interface {
	EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error
	UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error
	LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error
}

func Edit(log *slog.Logger, editor shutdownEditor, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

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

		storageReq := dto.EditShutdownRequest{
			OrganizationID:      req.OrganizationID,
			StartTime:           req.StartTime,
			EndTime:             req.EndTime,
			Reason:              req.Reason,
			GenerationLossMwh:   req.GenerationLossMwh,
			ReportedByContactID: req.ReportedByContactID,

			IdleDischargeVolumeThousandM3: req.IdleDischargeVolume,
			CreatedByUserID:               userID,
			FileIDs:                       fileIDs,
		}

		err = editor.EditShutdown(r.Context(), id, storageReq)
		if err != nil {
			// Cleanup uploaded files if update fails
			if uploadResult != nil {
				log.Warn("shutdown update failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
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

		// Update file links if explicitly requested
		if shouldUpdateFiles {
			// Remove old links
			if err := editor.UnlinkShutdownFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}

			// Add new links (if any)
			if len(fileIDs) > 0 {
				if err := editor.LinkShutdownFiles(r.Context(), id, fileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("shutdown updated successfully",
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

// parseMultipartEditRequest parses shutdown data from multipart form and handles file uploads
func parseMultipartEditRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
) (editRequest, *fileupload.UploadResult, error) {
	const op = "shutdowns.parseMultipartEditRequest"

	// Parse optional fields
	orgID, err := formparser.GetFormInt64(r, "organization_id")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	startTime, err := formparser.GetFormTime(r, "start_time", time.RFC3339)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid start_time format (use RFC3339): %w", err)
	}

	endTime, err := formparser.GetFormTime(r, "end_time", time.RFC3339)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid end_time format (use RFC3339): %w", err)
	}

	reason := formparser.GetFormString(r, "reason")

	generationLoss, err := formparser.GetFormFloat64(r, "generation_loss")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid generation_loss: %w", err)
	}

	reportedByContactID, err := formparser.GetFormInt64(r, "reported_by_contact_id")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid reported_by_contact_id: %w", err)
	}

	idleDischargeVolume, err := formparser.GetFormFloat64(r, "idle_discharge_volume")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid idle_discharge_volume: %w", err)
	}

	// Create request object
	req := editRequest{
		OrganizationID:      orgID,
		StartTime:           startTime,
		EndTime:             convertToDoublePointer(endTime),
		Reason:              reason,
		GenerationLossMwh:   generationLoss,
		ReportedByContactID: reportedByContactID,
		IdleDischargeVolume: idleDischargeVolume,
	}

	// Process file uploads (use current time as upload date for edits)
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		"shutdown",
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

// convertToDoublePointer converts *time.Time to **time.Time for nullable nullable fields
func convertToDoublePointer(t *time.Time) **time.Time {
	if t == nil {
		return nil
	}
	return &t
}
