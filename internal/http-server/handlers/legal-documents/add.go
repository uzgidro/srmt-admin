package legaldocuments

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addRequest struct {
	Name         string    `json:"name" validate:"required"`
	Number       *string   `json:"number,omitempty"`
	DocumentDate time.Time `json:"document_date" validate:"required"`
	TypeID       int       `json:"type_id" validate:"required,min=1"`
	FileIDs      []int64   `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type documentAdder interface {
	AddLegalDocument(ctx context.Context, req dto.AddLegalDocumentRequest, createdByID int64) (int64, error)
	LinkLegalDocumentFiles(ctx context.Context, documentID int64, fileIDs []int64) error
}

func Add(log *slog.Logger, adder documentAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.legal-document.add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Create DTO for storage
		storageReq := dto.AddLegalDocumentRequest{
			Name:         req.Name,
			Number:       req.Number,
			DocumentDate: req.DocumentDate,
			TypeID:       req.TypeID,
			FileIDs:      req.FileIDs,
		}

		// Create document
		id, err := adder.AddLegalDocument(r.Context(), storageReq, userID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation", slog.Int("type_id", req.TypeID))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid type ID"))
				return
			}
			log.Error("failed to add legal document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add legal document"))
			return
		}

		// Link files if provided
		if len(req.FileIDs) > 0 {
			if err := adder.LinkLegalDocumentFiles(r.Context(), id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
				// Don't fail the request, just log the error
			}
		}

		log.Info("legal document added successfully",
			slog.Int64("id", id),
			slog.Int("total_files", len(req.FileIDs)),
		)

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{
			Response: resp.Created(),
			ID:       id,
		})
	}
}
