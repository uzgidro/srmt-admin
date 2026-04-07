package infraevent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addRequest struct {
	CategoryID     int64   `json:"category_id" validate:"required"`
	OrganizationID int64   `json:"organization_id" validate:"required"`
	OccurredAt     string  `json:"occurred_at" validate:"required"`
	RestoredAt     *string `json:"restored_at,omitempty"`
	Description    string  `json:"description" validate:"required"`
	Remediation    *string `json:"remediation,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	FileIDs        []int64 `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type eventAdder interface {
	CreateInfraEvent(ctx context.Context, req dto.AddInfraEventRequest) (int64, error)
	LinkInfraEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error
}

func Create(log *slog.Logger, adder eventAdder, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event.create"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
		var fileIDs []int64
		var uploadResult *fileupload.UploadResult
		var occurredAt time.Time

		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			req, occurredAt, uploadResult, err = parseMultipartAddRequest(r, log, uploader, saver, categoryGetter)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
			fileIDs = append(existingFileIDs, uploadResult.FileIDs...)
		} else {
			log.Info("processing application/json request")

			if err := render.DecodeJSON(r.Body, &req); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			occurredAt, err = time.Parse(time.RFC3339, req.OccurredAt)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'occurred_at' format, use ISO 8601"))
				return
			}

			fileIDs = req.FileIDs
		}

		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)

			if uploadResult != nil {
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		var restoredAt *time.Time
		if req.RestoredAt != nil {
			t, err := time.Parse(time.RFC3339, *req.RestoredAt)
			if err != nil {
				if uploadResult != nil {
					fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
				}
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'restored_at' format, use ISO 8601"))
				return
			}
			restoredAt = &t
		}

		id, err := adder.CreateInfraEvent(r.Context(), dto.AddInfraEventRequest{
			CategoryID:      req.CategoryID,
			OrganizationID:  req.OrganizationID,
			OccurredAt:      occurredAt,
			RestoredAt:      restoredAt,
			Description:     req.Description,
			Remediation:     req.Remediation,
			Notes:           req.Notes,
			CreatedByUserID: userID,
		})
		if err != nil {
			if uploadResult != nil {
				log.Warn("event creation failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid category_id or organization_id"))
				return
			}
			if errors.Is(err, storage.ErrCheckConstraintViolation) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("restored_at must be after occurred_at"))
				return
			}
			log.Error("failed to create infra event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create event"))
			return
		}

		if len(fileIDs) > 0 {
			if err := adder.LinkInfraEventFiles(r.Context(), id, fileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}

		log.Info("infra event created", slog.Int64("id", id), slog.Int("total_files", len(fileIDs)))
		render.Status(r, http.StatusCreated)
		response := addResponse{
			Response: resp.OK(),
			ID:       id,
		}
		if uploadResult != nil && len(uploadResult.UploadedFiles) > 0 {
			response.UploadedFiles = uploadResult.UploadedFiles
		}
		render.JSON(w, r, response)
	}
}

func parseMultipartAddRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (addRequest, time.Time, *fileupload.UploadResult, error) {
	const op = "infraevent.parseMultipartAddRequest"

	categoryID, err := formparser.GetFormInt64Required(r, "category_id")
	if err != nil {
		return addRequest{}, time.Time{}, nil, err
	}

	orgID, err := formparser.GetFormInt64Required(r, "organization_id")
	if err != nil {
		return addRequest{}, time.Time{}, nil, err
	}

	occurredAt, err := formparser.GetFormTimeRequired(r, "occurred_at", time.RFC3339)
	if err != nil {
		return addRequest{}, time.Time{}, nil, fmt.Errorf("invalid or missing occurred_at (use RFC3339 format): %w", err)
	}

	description, err := formparser.GetFormStringRequired(r, "description")
	if err != nil {
		return addRequest{}, time.Time{}, nil, err
	}

	remediation := formparser.GetFormString(r, "remediation")
	notes := formparser.GetFormString(r, "notes")

	restoredAtStr := formparser.GetFormString(r, "restored_at")

	req := addRequest{
		CategoryID:     categoryID,
		OrganizationID: orgID,
		OccurredAt:     occurredAt.Format(time.RFC3339),
		Description:    description,
		Remediation:    remediation,
		Notes:          notes,
		RestoredAt:     restoredAtStr,
	}

	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(), r, log, uploader, saver, categoryGetter,
		"infra-events", "Инфра события", occurredAt,
	)
	if err != nil {
		return addRequest{}, time.Time{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart form parsed successfully", slog.Int("uploaded_files", len(uploadResult.FileIDs)))

	return req, occurredAt, uploadResult, nil
}
