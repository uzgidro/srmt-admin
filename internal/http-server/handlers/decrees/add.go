package decrees

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"

	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type decreeAdder interface {
	AddDecree(ctx context.Context, req dto.AddDecreeRequest, files []*multipart.FileHeader, createdByID int64) (int64, error)
}

func Add(log *slog.Logger, adder decreeAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.decrees.add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req dto.AddDecreeRequest
		var files []*multipart.FileHeader

		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			// Set max size for parsing
			err := r.ParseMultipartForm(100 << 20) // 100MB
			if err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid form data"))
				return
			}

			req, files, err = parseMultipartAddRequest(r)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}
		} else {
			log.Info("processing application/json request")

			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Check if we need to map linked docs or anything else?
		// DTO matches service requirement, so we can pass it directly.

		id, err := adder.AddDecree(r.Context(), req, files, userID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid reference ID (type, status, contact, or organization)"))
				return
			}
			log.Error("failed to add decree", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add decree"))
			return
		}

		log.Info("decree added successfully", slog.Int64("id", id))

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{
			Response: resp.Created(),
			ID:       id,
		})
	}
}

func parseMultipartAddRequest(r *http.Request) (dto.AddDecreeRequest, []*multipart.FileHeader, error) {
	name, err := formparser.GetFormStringRequired(r, "name")
	if err != nil {
		return dto.AddDecreeRequest{}, nil, err
	}

	// Document date
	// Existing handler used "2006-01-02"
	documentDate, err := formparser.GetFormDateRequired(r, "document_date")
	if err != nil {
		return dto.AddDecreeRequest{}, nil, err
	}

	typeIDInt64, err := formparser.GetFormInt64Required(r, "type_id")
	if err != nil {
		return dto.AddDecreeRequest{}, nil, fmt.Errorf("invalid or missing type_id: %w", err)
	}

	req := dto.AddDecreeRequest{
		Name:         name,
		Number:       formparser.GetFormString(r, "number"),
		Description:  formparser.GetFormString(r, "description"),
		DocumentDate: documentDate,
		TypeID:       int(typeIDInt64),
	}

	// Optional fields
	if statusID, err := formparser.GetFormInt64(r, "status_id"); err == nil && statusID != nil {
		statusIDInt := int(*statusID)
		req.StatusID = &statusIDInt
	}
	if contactID, err := formparser.GetFormInt64(r, "responsible_contact_id"); err == nil {
		req.ResponsibleContactID = contactID
	}
	if orgID, err := formparser.GetFormInt64(r, "organization_id"); err == nil {
		req.OrganizationID = orgID
	}
	if executorID, err := formparser.GetFormInt64(r, "executor_contact_id"); err == nil {
		req.ExecutorContactID = executorID
	}
	if parentID, err := formparser.GetFormInt64(r, "parent_document_id"); err == nil {
		req.ParentDocumentID = parentID
	}

	if dueDate, err := formparser.GetFormDate(r, "due_date"); err != nil {
		return dto.AddDecreeRequest{}, nil, fmt.Errorf("invalid due_date: %w", err)
	} else if dueDate != nil {
		req.DueDate = dueDate
	}

	// Files
	var files []*multipart.FileHeader
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if fhs, ok := r.MultipartForm.File["files"]; ok {
			files = fhs
		}
	}

	existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
	req.FileIDs = existingFileIDs

	return req, files, nil
}
