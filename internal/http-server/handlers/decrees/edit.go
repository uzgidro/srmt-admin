package decrees

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"

	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type decreeEditor interface {
	EditDecree(ctx context.Context, id int64, req dto.EditDecreeRequest, files []*multipart.FileHeader, updatedByID int64) error
}

func Edit(log *slog.Logger, editor decreeEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.decrees.edit"
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

		var req dto.EditDecreeRequest
		var files []*multipart.FileHeader

		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			err := r.ParseMultipartForm(100 << 20)
			if err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid form data"))
				return
			}

			req, files, err = parseMultipartEditRequest(r)
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

		// Edit request doesn't have strict validation in the original handler
		// But we could add check for fields? DTO has `validate` tags now if we want to use them.
		// Original handler didn't use validator. Keep as is for backward compatibility or add it?
		// "All fields are pointers (optional)" - so validation might be loose.

		err = editor.EditDecree(r.Context(), id, req, files, userID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("decree not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Decree not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid reference ID"))
				return
			}
			log.Error("failed to update decree", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update decree"))
			return
		}

		log.Info("decree updated successfully", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func parseMultipartEditRequest(r *http.Request) (dto.EditDecreeRequest, []*multipart.FileHeader, error) {
	var req dto.EditDecreeRequest

	if name := formparser.GetFormString(r, "name"); name != nil {
		req.Name = name
	}
	if number := formparser.GetFormString(r, "number"); number != nil {
		req.Number = number
	}
	if description := formparser.GetFormString(r, "description"); description != nil {
		req.Description = description
	}

	if documentDate, err := formparser.GetFormTime(r, "document_date", "2006-01-02"); err != nil {
		return dto.EditDecreeRequest{}, nil, fmt.Errorf("invalid document_date format (use YYYY-MM-DD): %w", err)
	} else if documentDate != nil {
		req.DocumentDate = documentDate
	}

	if typeID, err := formparser.GetFormInt64(r, "type_id"); err == nil && typeID != nil {
		typeIDInt := int(*typeID)
		req.TypeID = &typeIDInt
	}
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

	if dueDate, err := formparser.GetFormTime(r, "due_date", "2006-01-02"); err != nil {
		return dto.EditDecreeRequest{}, nil, fmt.Errorf("invalid due_date format (use YYYY-MM-DD): %w", err)
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
