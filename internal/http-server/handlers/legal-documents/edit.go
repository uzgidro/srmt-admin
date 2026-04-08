package legaldocuments

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
	Name         *string    `json:"name,omitempty"`
	Number       *string    `json:"number,omitempty"`
	DocumentDate *time.Time `json:"document_date,omitempty"`
	TypeID       *int       `json:"type_id,omitempty"`
	FileIDs      []int64    `json:"file_ids,omitempty"`
}

type documentEditor interface {
	EditLegalDocument(ctx context.Context, id int64, req dto.EditLegalDocumentRequest, updatedByID int64) error
	UnlinkLegalDocumentFiles(ctx context.Context, documentID int64) error
	LinkLegalDocumentFiles(ctx context.Context, documentID int64, fileIDs []int64) error
}

func Edit(log *slog.Logger, editor documentEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.legal-document.edit"
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

		// Build storage request
		storageReq := dto.EditLegalDocumentRequest{
			Name:         req.Name,
			Number:       req.Number,
			DocumentDate: req.DocumentDate,
			TypeID:       req.TypeID,
			FileIDs:      req.FileIDs,
		}

		// Update document
		err = editor.EditLegalDocument(r.Context(), id, storageReq, userID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("legal document not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Legal document not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (invalid type_id)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid type ID"))
				return
			}
			log.Error("failed to update legal document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update legal document"))
			return
		}

		// Update file links if explicitly requested
		if req.FileIDs != nil {
			// Remove old links
			if err := editor.UnlinkLegalDocumentFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}

			// Add new links (if any)
			if len(req.FileIDs) > 0 {
				if err := editor.LinkLegalDocumentFiles(r.Context(), id, req.FileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("legal document updated successfully",
			slog.Int64("id", id),
			slog.Bool("files_updated", req.FileIDs != nil),
			slog.Int("total_files", len(req.FileIDs)),
		)

		render.JSON(w, r, resp.OK())
	}
}
