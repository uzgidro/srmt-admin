package add

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
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"time"
)

type Request struct {
	OrganizationID int64      `json:"organization_id" validate:"required"`
	StartedAt      time.Time  `json:"started_at" validate:"required"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	FlowRate       float64    `json:"flow_rate" validate:"required,gt=0"`
	Reason         *string    `json:"reason,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
}

type Response struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type DischargeAdder interface {
	AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error)
	LinkDischargeFiles(ctx context.Context, dischargeID int64, fileIDs []int64) error
}

func New(log *slog.Logger, adder DischargeAdder, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req Request
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

		// Create discharge
		id, err := adder.AddDischarge(r.Context(), req.OrganizationID, userID, req.StartedAt, req.EndedAt, req.FlowRate, req.Reason)
		if err != nil {
			// Cleanup uploaded files if discharge creation fails
			if uploadResult != nil {
				log.Warn("discharge creation failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("organization not found", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to add discharge", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add discharge"))
			return
		}

		// Link files if provided
		if len(fileIDs) > 0 {
			if err := adder.LinkDischargeFiles(r.Context(), id, fileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
				// Don't fail the request, just log the error
			}
		}

		log.Info("discharge added successfully",
			slog.Int64("id", id),
			slog.Int("total_files", len(fileIDs)),
			slog.Int("uploaded_files", len(uploadResult.FileIDs)),
		)

		render.Status(r, http.StatusCreated)
		response := Response{
			Response: resp.Created(),
			ID:       id,
		}
		if uploadResult != nil && len(uploadResult.UploadedFiles) > 0 {
			response.UploadedFiles = uploadResult.UploadedFiles
		}
		render.JSON(w, r, response)
	}
}

// parseMultipartAddRequest parses discharge data from multipart form and handles file uploads
func parseMultipartAddRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
) (Request, *fileupload.UploadResult, error) {
	const op = "discharge.add.parseMultipartAddRequest"

	// Parse organization_id (required)
	orgID, err := formparser.GetFormInt64Required(r, "organization_id")
	if err != nil {
		return Request{}, nil, err
	}

	// Parse started_at (required)
	startedAt, err := formparser.GetFormTimeRequired(r, "started_at", time.RFC3339)
	if err != nil {
		return Request{}, nil, fmt.Errorf("invalid or missing started_at (use RFC3339 format): %w", err)
	}

	// Parse ended_at (optional)
	endedAt, err := formparser.GetFormTime(r, "ended_at", time.RFC3339)
	if err != nil {
		return Request{}, nil, fmt.Errorf("invalid ended_at format (use RFC3339): %w", err)
	}

	// Parse flow_rate (required)
	flowRatePtr, err := formparser.GetFormFloat64(r, "flow_rate")
	if err != nil || flowRatePtr == nil {
		return Request{}, nil, fmt.Errorf("flow_rate is required and must be a valid number: %w", err)
	}
	flowRate := *flowRatePtr

	// Parse reason (optional)
	reason := formparser.GetFormString(r, "reason")

	// Create request object
	req := Request{
		OrganizationID: orgID,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		FlowRate:       flowRate,
		Reason:         reason,
	}

	// Process file uploads
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		"discharge", // category name for MinIO path
		startedAt,
	)
	if err != nil {
		return Request{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
