package infraevent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	CategoryID      *int64  `json:"category_id,omitempty"`
	OrganizationID  *int64  `json:"organization_id,omitempty"`
	OccurredAt      *string `json:"occurred_at,omitempty"`
	RestoredAt      *string `json:"restored_at,omitempty"`
	ClearRestoredAt *bool   `json:"clear_restored_at,omitempty"`
	Description     *string `json:"description,omitempty"`
	Remediation     *string `json:"remediation,omitempty"`
	Notes           *string `json:"notes,omitempty"`
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type eventEditor interface {
	UpdateInfraEvent(ctx context.Context, id int64, req dto.EditInfraEventRequest) error
	UnlinkInfraEventFiles(ctx context.Context, eventID int64) error
	LinkInfraEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error
}

func Update(log *slog.Logger, editor eventEditor, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event.update"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editRequest
		var fileIDs []int64
		var shouldUpdateFiles bool
		var uploadResult *fileupload.UploadResult

		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			req, uploadResult, err = parseMultipartEditRequest(r, log, uploader, saver, categoryGetter)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			if formparser.HasFormField(r, "file_ids") || len(uploadResult.FileIDs) > 0 {
				shouldUpdateFiles = true
				existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
				fileIDs = append(existingFileIDs, uploadResult.FileIDs...)
			}
		} else {
			log.Info("processing application/json request")

			if err := render.DecodeJSON(r.Body, &req); err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			if req.FileIDs != nil {
				shouldUpdateFiles = true
				fileIDs = req.FileIDs
			}
		}

		storageReq := dto.EditInfraEventRequest{
			CategoryID:     req.CategoryID,
			OrganizationID: req.OrganizationID,
			Description:    req.Description,
			Remediation:    req.Remediation,
			Notes:          req.Notes,
		}

		if req.ClearRestoredAt != nil && *req.ClearRestoredAt {
			storageReq.ClearRestoredAt = true
		}

		if req.OccurredAt != nil {
			t, err := time.Parse(time.RFC3339, *req.OccurredAt)
			if err != nil {
				if uploadResult != nil {
					fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
				}
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'occurred_at' format, use ISO 8601"))
				return
			}
			storageReq.OccurredAt = &t
		}

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
			storageReq.RestoredAt = &t
		}

		err = editor.UpdateInfraEvent(r.Context(), id, storageReq)
		if err != nil {
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
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid category_id or organization_id"))
				return
			}
			if errors.Is(err, storage.ErrCheckConstraintViolation) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("restored_at must be after occurred_at"))
				return
			}
			log.Error("failed to update infra event", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update event"))
			return
		}

		if shouldUpdateFiles {
			if err := editor.UnlinkInfraEventFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink files", sl.Err(err))
			}
			if len(fileIDs) > 0 {
				if err := editor.LinkInfraEventFiles(r.Context(), id, fileIDs); err != nil {
					log.Error("failed to link files", sl.Err(err))
				}
			}
		}

		log.Info("infra event updated",
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

func parseMultipartEditRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (editRequest, *fileupload.UploadResult, error) {
	const op = "infraevent.parseMultipartEditRequest"

	categoryID, err := formparser.GetFormInt64(r, "category_id")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid category_id: %w", err)
	}

	orgID, err := formparser.GetFormInt64(r, "organization_id")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid organization_id: %w", err)
	}

	occurredAt, err := formparser.GetFormTime(r, "occurred_at", time.RFC3339)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid occurred_at format (use RFC3339): %w", err)
	}

	restoredAt := formparser.GetFormString(r, "restored_at")
	description := formparser.GetFormString(r, "description")
	remediation := formparser.GetFormString(r, "remediation")
	notes := formparser.GetFormString(r, "notes")

	req := editRequest{
		CategoryID:     categoryID,
		OrganizationID: orgID,
		Description:    description,
		Remediation:    remediation,
		Notes:          notes,
		RestoredAt:     restoredAt,
	}

	if occurredAt != nil {
		s := occurredAt.Format(time.RFC3339)
		req.OccurredAt = &s
	}

	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(), r, log, uploader, saver, categoryGetter,
		"infra-events", "Инфра события", time.Now(),
	)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart edit form parsed successfully", slog.Int("uploaded_files", len(uploadResult.FileIDs)))

	return req, uploadResult, nil
}
