package edit

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
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"
	"time"
)

type Request struct {
	StartedAt      *time.Time `json:"started_at,omitempty"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	FlowRate       *float64   `json:"flow_rate,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	Approved       *bool      `json:"approved,omitempty"`
	OrganizationID *int64     `json:"organization_id,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
}

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type DischargeEditor interface {
	EditDischarge(ctx context.Context, id, approvedByID int64, startTime, endTime *time.Time, flowRate *float64, reason *string, approved *bool, organizationID *int64) error
	UnlinkDischargeFiles(ctx context.Context, dischargeID int64) error
	LinkDischargeFiles(ctx context.Context, dischargeID int64, fileIDs []int64) error
}

func New(log *slog.Logger, editor DischargeEditor, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.patch.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		dischargeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid discharge ID format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid discharge ID"))
			return
		}

		var req Request
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
				log.Error("failed to decode request body", sl.Err(err))
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

		err = editor.EditDischarge(r.Context(), dischargeID, userID, req.StartedAt, req.EndedAt, req.FlowRate, req.Reason, req.Approved, req.OrganizationID)
		if err != nil {
			// Cleanup uploaded files if update fails
			if uploadResult != nil {
				log.Warn("discharge update failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("discharge not found", "id", dischargeID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Discharge not found"))
				return
			}
			log.Error("failed to edit discharge", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to edit discharge"))
			return
		}

		// Update file links if explicitly requested
		if shouldUpdateFiles {
			// Remove old links
			if err := editor.UnlinkDischargeFiles(r.Context(), dischargeID); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}

			// Add new links (if any)
			if len(fileIDs) > 0 {
				if err := editor.LinkDischargeFiles(r.Context(), dischargeID, fileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("discharge updated successfully",
			slog.Int64("id", dischargeID),
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

// parseMultipartEditRequest parses discharge data from multipart form and handles file uploads
func parseMultipartEditRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
) (Request, *fileupload.UploadResult, error) {
	const op = "discharge.edit.parseMultipartEditRequest"

	// Parse optional fields
	orgID, err := formparser.GetFormInt64(r, "organization_id")
	if err != nil {
		return Request{}, nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	startedAt, err := formparser.GetFormTime(r, "started_at", time.RFC3339)
	if err != nil {
		return Request{}, nil, fmt.Errorf("invalid started_at format (use RFC3339): %w", err)
	}

	endedAt, err := formparser.GetFormTime(r, "ended_at", time.RFC3339)
	if err != nil {
		return Request{}, nil, fmt.Errorf("invalid ended_at format (use RFC3339): %w", err)
	}

	flowRate, err := formparser.GetFormFloat64(r, "flow_rate")
	if err != nil {
		return Request{}, nil, fmt.Errorf("invalid flow_rate: %w", err)
	}

	reason := formparser.GetFormString(r, "reason")
	approved, err := formparser.GetFormBool(r, "approved")
	if err != nil {
		return Request{}, nil, fmt.Errorf("invalid approved: %w", err)
	}

	// Create request object
	req := Request{
		OrganizationID: orgID,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		FlowRate:       flowRate,
		Reason:         reason,
		Approved:       approved,
	}

	// Process file uploads (use current time as upload date for edits)
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		"discharge",
		time.Now(), // For edits, use current time
	)
	if err != nil {
		return Request{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart edit form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
