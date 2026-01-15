package reports

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
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	Name                 *string                     `json:"name,omitempty"`
	Number               *string                     `json:"number,omitempty"`
	DocumentDate         *time.Time                  `json:"document_date,omitempty"`
	Description          *string                     `json:"description,omitempty"`
	TypeID               *int                        `json:"type_id,omitempty"`
	StatusID             *int                        `json:"status_id,omitempty"`
	ResponsibleContactID *int64                      `json:"responsible_contact_id,omitempty"`
	OrganizationID       *int64                      `json:"organization_id,omitempty"`
	ExecutorContactID    *int64                      `json:"executor_contact_id,omitempty"`
	DueDate              *time.Time                  `json:"due_date,omitempty"`
	ParentDocumentID     *int64                      `json:"parent_document_id,omitempty"`
	FileIDs              []int64                     `json:"file_ids,omitempty"`
	LinkedDocuments      []dto.LinkedDocumentRequest `json:"linked_documents,omitempty"`
}

type reportEditor interface {
	EditReport(ctx context.Context, id int64, req dto.EditReportRequest, updatedByID int64) error
	UnlinkReportFiles(ctx context.Context, reportID int64) error
	LinkReportFiles(ctx context.Context, reportID int64, fileIDs []int64) error
	UnlinkReportDocuments(ctx context.Context, reportID int64) error
	LinkReportDocuments(ctx context.Context, reportID int64, links []dto.LinkedDocumentRequest, userID int64) error
}

func Edit(log *slog.Logger, editor reportEditor, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reports.edit"
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
		var uploadResult *fileupload.UploadResult
		var hasFileChanges bool

		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			req, uploadResult, err = parseMultipartEditRequest(r, log, uploader, saver, categoryGetter)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
			fileIDs = append(existingFileIDs, uploadResult.FileIDs...)
			hasFileChanges = true
		} else {
			log.Info("processing application/json request")

			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			if req.FileIDs != nil {
				fileIDs = req.FileIDs
				hasFileChanges = true
			}
		}

		storageReq := dto.EditReportRequest{
			Name:                 req.Name,
			Number:               req.Number,
			DocumentDate:         req.DocumentDate,
			Description:          req.Description,
			TypeID:               req.TypeID,
			StatusID:             req.StatusID,
			ResponsibleContactID: req.ResponsibleContactID,
			OrganizationID:       req.OrganizationID,
			ExecutorContactID:    req.ExecutorContactID,
			DueDate:              req.DueDate,
			ParentDocumentID:     req.ParentDocumentID,
			FileIDs:              fileIDs,
			LinkedDocuments:      req.LinkedDocuments,
		}

		err = editor.EditReport(r.Context(), id, storageReq, userID)
		if err != nil {
			if uploadResult != nil {
				log.Warn("report update failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("report not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Report not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid reference ID"))
				return
			}
			log.Error("failed to update report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update report"))
			return
		}

		// Update file links if provided
		if hasFileChanges {
			if err := editor.UnlinkReportFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink files", sl.Err(err))
			}
			if len(fileIDs) > 0 {
				if err := editor.LinkReportFiles(r.Context(), id, fileIDs); err != nil {
					log.Error("failed to link files", sl.Err(err))
				}
			}
		}

		// Update document links if provided
		if req.LinkedDocuments != nil {
			if err := editor.UnlinkReportDocuments(r.Context(), id); err != nil {
				log.Error("failed to unlink documents", sl.Err(err))
			}
			if len(req.LinkedDocuments) > 0 {
				if err := editor.LinkReportDocuments(r.Context(), id, req.LinkedDocuments, userID); err != nil {
					log.Error("failed to link documents", sl.Err(err))
				}
			}
		}

		log.Info("report updated successfully", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func parseMultipartEditRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (editRequest, *fileupload.UploadResult, error) {
	const op = "reports.parseMultipartEditRequest"

	var req editRequest

	if name := formparser.GetFormString(r, "name"); name != nil {
		req.Name = name
	}
	if number := formparser.GetFormString(r, "number"); number != nil {
		req.Number = number
	}
	if description := formparser.GetFormString(r, "description"); description != nil {
		req.Description = description
	}

	if documentDateStr := formparser.GetFormString(r, "document_date"); documentDateStr != nil {
		documentDate, err := time.Parse("2006-01-02", *documentDateStr)
		if err != nil {
			return editRequest{}, nil, fmt.Errorf("invalid document_date format (use YYYY-MM-DD): %w", err)
		}
		req.DocumentDate = &documentDate
	}

	if typeID, err := formparser.GetFormInt64(r, "type_id"); err == nil && typeID != nil {
		typeIDInt := int(*typeID)
		req.TypeID = &typeIDInt
	}
	if statusID, err := formparser.GetFormInt64(r, "status_id"); err == nil && statusID != nil {
		statusIDInt := int(*statusID)
		req.StatusID = &statusIDInt
	}
	if responsibleContactID, err := formparser.GetFormInt64(r, "responsible_contact_id"); err == nil {
		req.ResponsibleContactID = responsibleContactID
	}
	if organizationID, err := formparser.GetFormInt64(r, "organization_id"); err == nil {
		req.OrganizationID = organizationID
	}
	if executorContactID, err := formparser.GetFormInt64(r, "executor_contact_id"); err == nil {
		req.ExecutorContactID = executorContactID
	}
	if parentDocumentID, err := formparser.GetFormInt64(r, "parent_document_id"); err == nil {
		req.ParentDocumentID = parentDocumentID
	}

	if dueDateStr := formparser.GetFormString(r, "due_date"); dueDateStr != nil {
		dueDate, err := time.Parse("2006-01-02", *dueDateStr)
		if err != nil {
			return editRequest{}, nil, fmt.Errorf("invalid due_date format (use YYYY-MM-DD): %w", err)
		}
		req.DueDate = &dueDate
	}

	targetDate := time.Now()
	if req.DocumentDate != nil {
		targetDate = *req.DocumentDate
	}

	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		categoryGetter,
		"reports",
		"Рапорты",
		targetDate,
	)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	return req, uploadResult, nil
}
