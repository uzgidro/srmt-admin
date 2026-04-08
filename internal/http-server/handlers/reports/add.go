package reports

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
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
	ID int64 `json:"id"`
}

type reportAdder interface {
	AddReport(ctx context.Context, req dto.AddReportRequest, createdByID int64) (int64, error)
	LinkReportFiles(ctx context.Context, reportID int64, fileIDs []int64) error
	LinkReportDocuments(ctx context.Context, reportID int64, links []dto.LinkedDocumentRequest, userID int64) error
}

func Add(log *slog.Logger, adder reportAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reports.add"
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

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		storageReq := dto.AddReportRequest{
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

		id, err := adder.AddReport(r.Context(), storageReq, userID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid reference ID (type, status, contact, or organization)"))
				return
			}
			log.Error("failed to add report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add report"))
			return
		}

		if len(req.FileIDs) > 0 {
			if err := adder.LinkReportFiles(r.Context(), id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
			}
		}

		if len(req.LinkedDocuments) > 0 {
			if err := adder.LinkReportDocuments(r.Context(), id, req.LinkedDocuments, userID); err != nil {
				log.Error("failed to link documents", sl.Err(err))
			}
		}

		log.Info("report added successfully",
			slog.Int64("id", id),
			slog.Int("files", len(req.FileIDs)),
		)

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{
			Response: resp.Created(),
			ID:       id,
		})
	}
}
