package shutdowns

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"time"
)

type addRequest struct {
	OrganizationID      int64      `json:"organization_id" validate:"required"`
	StartTime           time.Time  `json:"start_time" validate:"required"`
	EndTime             *time.Time `json:"end_time,omitempty"`
	Reason              *string    `json:"reason,omitempty"`
	GenerationLossMwh   *float64   `json:"generation_loss,omitempty"`
	ReportedByContactID *int64     `json:"reported_by_contact_id,omitempty"`

	IdleDischargeVolume *float64 `json:"idle_discharge_volume,omitempty"`
	FileIDs             []int64  `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type ShutdownAdder interface {
	AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error)
	LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error
}

func Add(log *slog.Logger, adder ShutdownAdder, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Add"
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
			req, uploadResult, err = parseMultipartAddRequest(r, log, uploader, saver)
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

		if req.IdleDischargeVolume != nil && req.EndTime == nil {
			// Cleanup uploaded files if validation fails
			if uploadResult != nil {
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			log.Warn("validation failed: end_time is required if idle_discharge_volume is provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_time is required when providing idle_discharge_volume"))
			return
		}
		if req.EndTime != nil && !req.EndTime.After(req.StartTime) {
			// Cleanup uploaded files if validation fails
			if uploadResult != nil {
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			log.Warn("validation failed: end_time must be after start_time")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_time must be after start_time"))
			return
		}

		storageReq := dto.AddShutdownRequest{
			OrganizationID:      req.OrganizationID,
			StartTime:           req.StartTime,
			EndTime:             req.EndTime,
			Reason:              req.Reason,
			GenerationLossMwh:   req.GenerationLossMwh,
			ReportedByContactID: req.ReportedByContactID,
			CreatedByUserID:     userID,

			IdleDischargeVolumeThousandM3: req.IdleDischargeVolume,
			FileIDs:                       fileIDs,
		}

		id, err := adder.AddShutdown(r.Context(), storageReq)
		if err != nil {
			// Cleanup uploaded files if shutdown creation fails
			if uploadResult != nil {
				log.Warn("shutdown creation failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation (org or contact not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id or reported_by_contact_id"))
				return
			}
			log.Error("failed to add shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add shutdown"))
			return
		}

		// Link files if provided
		if len(fileIDs) > 0 {
			if err := adder.LinkShutdownFiles(r.Context(), id, fileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
				// Don't fail the request, just log the error
			}
		}

		uploadedFilesCount := 0
		if uploadResult != nil {
			uploadedFilesCount = len(uploadResult.FileIDs)
		}
		log.Info("shutdown added successfully",
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

// parseMultipartAddRequest parses shutdown data from multipart form and handles file uploads
func parseMultipartAddRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
) (addRequest, *fileupload.UploadResult, error) {
	const op = "shutdowns.parseMultipartAddRequest"

	// Parse organization_id (required)
	orgID, err := formparser.GetFormInt64Required(r, "organization_id")
	if err != nil {
		return addRequest{}, nil, err
	}

	// Parse start_time (required)
	startTime, err := formparser.GetFormTimeRequired(r, "start_time", time.RFC3339)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid or missing start_time (use RFC3339 format): %w", err)
	}

	// Parse end_time (optional)
	endTime, err := formparser.GetFormTime(r, "end_time", time.RFC3339)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid end_time format (use RFC3339): %w", err)
	}

	// Parse reason (optional)
	reason := formparser.GetFormString(r, "reason")

	// Parse generation_loss (optional)
	generationLoss, err := formparser.GetFormFloat64(r, "generation_loss")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid generation_loss: %w", err)
	}

	// Parse reported_by_contact_id (optional)
	reportedByContactID, err := formparser.GetFormInt64(r, "reported_by_contact_id")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid reported_by_contact_id: %w", err)
	}

	// Parse idle_discharge_volume (optional)
	idleDischargeVolume, err := formparser.GetFormFloat64(r, "idle_discharge_volume")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid idle_discharge_volume: %w", err)
	}

	// Create request object
	req := addRequest{
		OrganizationID:      orgID,
		StartTime:           startTime,
		EndTime:             endTime,
		Reason:              reason,
		GenerationLossMwh:   generationLoss,
		ReportedByContactID: reportedByContactID,
		IdleDischargeVolume: idleDischargeVolume,
	}

	// Process file uploads
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		"shutdown", // category name for MinIO path
		startTime,
	)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
