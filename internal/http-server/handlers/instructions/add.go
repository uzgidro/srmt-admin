package instructions

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
	Name                 string                      `json:"name" validate:"required"`
	Number               *string                     `json:"number,omitempty"`
	DocumentDate         time.Time                   `json:"document_date" validate:"required"`
	Description          *string                     `json:"description,omitempty"`
	TypeID               int                         `json:"type_id" validate:"required,min=1"`
	StatusID             *int                        `json:"status_id,omitempty"`
	ResponsibleContactID *int64                      `json:"responsible_contact_id,omitempty"`
	OrganizationID       *int64                      `json:"organization_id,omitempty"`
	ExecutorContactID    *int64                      `json:"executor_contact_id,omitempty"`
	DueDate              *time.Time                  `json:"due_date,omitempty"`
	ParentDocumentID     *int64                      `json:"parent_document_id,omitempty"`
	FileIDs              []int64                     `json:"file_ids,omitempty"`
	LinkedDocuments      []dto.LinkedDocumentRequest `json:"linked_documents,omitempty"`
}

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type instructionAdder interface {
	AddInstruction(ctx context.Context, req dto.AddInstructionRequest, createdByID int64) (int64, error)
	LinkInstructionFiles(ctx context.Context, instructionID int64, fileIDs []int64) error
	LinkInstructionDocuments(ctx context.Context, instructionID int64, links []dto.LinkedDocumentRequest, userID int64) error
}

func Add(log *slog.Logger, adder instructionAdder, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.instructions.add"
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

		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			req, uploadResult, err = parseMultipartAddRequest(r, log, uploader, saver, categoryGetter)
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
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			fileIDs = req.FileIDs
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))

			if uploadResult != nil {
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		storageReq := dto.AddInstructionRequest{
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

		id, err := adder.AddInstruction(r.Context(), storageReq, userID)
		if err != nil {
			if uploadResult != nil {
				log.Warn("instruction creation failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid reference ID (type, status, contact, or organization)"))
				return
			}
			log.Error("failed to add instruction", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add instruction"))
			return
		}

		if len(fileIDs) > 0 {
			if err := adder.LinkInstructionFiles(r.Context(), id, fileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}

		if len(req.LinkedDocuments) > 0 {
			if err := adder.LinkInstructionDocuments(r.Context(), id, req.LinkedDocuments, userID); err != nil {
				log.Error("failed to link documents", sl.Err(err))
			}
		}

		uploadedFilesCount := 0
		if uploadResult != nil {
			uploadedFilesCount = len(uploadResult.FileIDs)
		}
		log.Info("instruction added successfully",
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

func parseMultipartAddRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (addRequest, *fileupload.UploadResult, error) {
	const op = "instructions.parseMultipartAddRequest"

	name, err := formparser.GetFormStringRequired(r, "name")
	if err != nil {
		return addRequest{}, nil, err
	}

	number := formparser.GetFormString(r, "number")
	description := formparser.GetFormString(r, "description")

	documentDate, err := formparser.GetFormDateRequired(r, "document_date")
	if err != nil {
		return addRequest{}, nil, err
	}

	typeIDInt64, err := formparser.GetFormInt64Required(r, "type_id")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid or missing type_id: %w", err)
	}
	typeID := int(typeIDInt64)

	req := addRequest{
		Name:         name,
		Number:       number,
		Description:  description,
		DocumentDate: documentDate,
		TypeID:       typeID,
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

	if dueDate, err := formparser.GetFormDate(r, "due_date"); err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid due_date: %w", err)
	} else if dueDate != nil {
		req.DueDate = dueDate
	}

	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		categoryGetter,
		"instructions",
		"Инструкции",
		documentDate,
	)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
