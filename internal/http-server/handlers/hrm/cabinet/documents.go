package cabinet

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// DocumentRepository defines the interface for document operations
type DocumentRepository interface {
	GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error)
	GetPersonnelDocuments(ctx context.Context, filter hrm.PersonnelDocumentFilter) ([]*hrmmodel.PersonnelDocument, error)
	GetPersonnelDocumentByID(ctx context.Context, id int64) (*hrmmodel.PersonnelDocument, error)
}

// GetMyDocuments returns documents for the currently authenticated employee
func GetMyDocuments(log *slog.Logger, repo DocumentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMyDocuments"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Find employee by user_id
		employee, err := repo.GetEmployeeByUserID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("employee not found", slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Employee profile not found"))
				return
			}
			log.Error("failed to get employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee"))
			return
		}

		// Parse query params for filtering
		filter := hrm.PersonnelDocumentFilter{
			EmployeeID: &employee.ID,
		}

		// Optional document type filter
		if docType := r.URL.Query().Get("type"); docType != "" {
			filter.DocumentType = &docType
		}

		// Get documents
		documents, err := repo.GetPersonnelDocuments(r.Context(), filter)
		if err != nil {
			log.Error("failed to get documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve documents"))
			return
		}

		// Build response
		response := make([]hrm.MyDocumentResponse, 0, len(documents))
		for _, d := range documents {
			status := "active"
			if d.IsVerified {
				status = "verified"
			}

			// Use document number as title
			title := d.DocumentType
			if d.DocumentNumber != nil {
				title = d.DocumentType + " " + *d.DocumentNumber
			}

			doc := hrm.MyDocumentResponse{
				ID:           d.ID,
				DocumentType: d.DocumentType,
				Title:        title,
				Status:       status,
				IssuedDate:   d.IssuedDate,
				ExpiryDate:   d.ExpiryDate,
				FileID:       d.FileID,
				CreatedAt:    d.CreatedAt,
			}

			response = append(response, doc)
		}

		log.Info("documents retrieved", slog.Int64("employee_id", employee.ID), slog.Int("count", len(documents)))
		render.JSON(w, r, response)
	}
}

// DownloadMyDocument provides a download link for a specific document
func DownloadMyDocument(log *slog.Logger, repo DocumentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.DownloadMyDocument"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Find employee by user_id
		employee, err := repo.GetEmployeeByUserID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("employee not found", slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Employee profile not found"))
				return
			}
			log.Error("failed to get employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee"))
			return
		}

		// Get document ID from URL
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid id parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// Get document
		document, err := repo.GetPersonnelDocumentByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("document not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document not found"))
				return
			}
			log.Error("failed to get document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve document"))
			return
		}

		// Verify ownership
		if document.EmployeeID != employee.ID {
			log.Warn("document does not belong to employee", slog.Int64("document_id", id), slog.Int64("employee_id", employee.ID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("You can only access your own documents"))
			return
		}

		// Check if file exists
		if document.FileID == nil {
			log.Warn("document has no file attached", slog.Int64("id", id))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.NotFound("No file attached to this document"))
			return
		}

		// Build title from document type and number
		title := document.DocumentType
		if document.DocumentNumber != nil {
			title = document.DocumentType + " " + *document.DocumentNumber
		}

		// Return file reference for download
		// In a full implementation, this would generate a presigned URL or redirect to file service
		response := struct {
			FileID     int64  `json:"file_id"`
			Title      string `json:"title"`
			DocumentID int64  `json:"document_id"`
		}{
			FileID:     *document.FileID,
			Title:      title,
			DocumentID: document.ID,
		}

		log.Info("document download requested", slog.Int64("employee_id", employee.ID), slog.Int64("document_id", id))
		render.JSON(w, r, response)
	}
}
