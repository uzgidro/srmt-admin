package decrees

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
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

type decreeEditor interface {
	EditDecree(ctx context.Context, id int64, req dto.EditDecreeRequest, updatedByID int64) error
	UnlinkDecreeFiles(ctx context.Context, decreeID int64) error
	LinkDecreeFiles(ctx context.Context, decreeID int64, fileIDs []int64) error
	UnlinkDecreeDocuments(ctx context.Context, decreeID int64) error
	LinkDecreeDocuments(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error
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

		var req editRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		storageReq := dto.EditDecreeRequest{
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
			FileIDs:              req.FileIDs,
			LinkedDocuments:      req.LinkedDocuments,
		}

		err = editor.EditDecree(r.Context(), id, storageReq, userID)
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

		if req.FileIDs != nil {
			if err := editor.UnlinkDecreeFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink files", sl.Err(err))
			}
			if len(req.FileIDs) > 0 {
				if err := editor.LinkDecreeFiles(r.Context(), id, req.FileIDs); err != nil {
					log.Error("failed to link files", sl.Err(err))
				}
			}
		}

		if req.LinkedDocuments != nil {
			if err := editor.UnlinkDecreeDocuments(r.Context(), id); err != nil {
				log.Error("failed to unlink documents", sl.Err(err))
			}
			if len(req.LinkedDocuments) > 0 {
				if err := editor.LinkDecreeDocuments(r.Context(), id, req.LinkedDocuments, userID); err != nil {
					log.Error("failed to link documents", sl.Err(err))
				}
			}
		}

		log.Info("decree updated successfully", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
