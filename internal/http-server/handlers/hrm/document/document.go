package document

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Repository Interfaces ---

type DocumentTypeRepository interface {
	AddDocumentType(ctx context.Context, req hrm.AddDocumentTypeRequest) (int, error)
	GetDocumentTypeByID(ctx context.Context, id int) (*hrmmodel.DocumentType, error)
	GetDocumentTypes(ctx context.Context, activeOnly bool) ([]*hrmmodel.DocumentType, error)
	EditDocumentType(ctx context.Context, id int, req hrm.EditDocumentTypeRequest) error
	DeleteDocumentType(ctx context.Context, id int) error
}

type DocumentRepository interface {
	AddDocument(ctx context.Context, req hrm.AddDocumentRequest, createdBy *int64) (int64, error)
	GetDocumentByID(ctx context.Context, id int64) (*hrmmodel.Document, error)
	GetDocuments(ctx context.Context, filter hrm.DocumentFilter) ([]*hrmmodel.Document, error)
	EditDocument(ctx context.Context, id int64, req hrm.EditDocumentRequest) error
	DeleteDocument(ctx context.Context, id int64) error
}

type SignatureRepository interface {
	AddDocumentSignature(ctx context.Context, req hrm.AddSignatureRequest) (int64, error)
	GetHRMDocumentSignatures(ctx context.Context, filter hrm.SignatureFilter) ([]*hrmmodel.DocumentSignature, error)
	SignHRMDocument(ctx context.Context, signatureID int64, signed bool, reason *string, ip string) error
}

type TemplateRepository interface {
	AddDocumentTemplate(ctx context.Context, req hrm.AddDocumentTemplateRequest, createdBy *int64) (int64, error)
	GetDocumentTemplateByID(ctx context.Context, id int64) (*hrmmodel.DocumentTemplate, error)
	GetDocumentTemplates(ctx context.Context, filter hrm.DocumentTemplateFilter) ([]*hrmmodel.DocumentTemplate, error)
	EditDocumentTemplate(ctx context.Context, id int64, req hrm.EditDocumentTemplateRequest) error
	DeleteDocumentTemplate(ctx context.Context, id int64) error
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Document Type Handlers ---

func GetTypes(log *slog.Logger, repo DocumentTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetTypes"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		activeOnly := r.URL.Query().Get("active_only") == "true"

		types, err := repo.GetDocumentTypes(r.Context(), activeOnly)
		if err != nil {
			log.Error("failed to get document types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve document types"))
			return
		}

		log.Info("successfully retrieved document types", slog.Int("count", len(types)))
		render.JSON(w, r, types)
	}
}

func GetTypeByID(log *slog.Logger, repo DocumentTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetTypeByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		docType, err := repo.GetDocumentTypeByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("document type not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document type not found"))
				return
			}
			log.Error("failed to get document type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve document type"))
			return
		}

		render.JSON(w, r, docType)
	}
}

func AddType(log *slog.Logger, repo DocumentTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.AddType"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddDocumentTypeRequest
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

		id, err := repo.AddDocumentType(r.Context(), req)
		if err != nil {
			log.Error("failed to add document type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add document type"))
			return
		}

		log.Info("document type added", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: int64(id)})
	}
}

func EditType(log *slog.Logger, repo DocumentTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.EditType"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditDocumentTypeRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditDocumentType(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("document type not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document type not found"))
				return
			}
			log.Error("failed to update document type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update document type"))
			return
		}

		log.Info("document type updated", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteType(log *slog.Logger, repo DocumentTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.DeleteType"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteDocumentType(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("document type not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document type not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("document type has dependencies", slog.Int("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: document type is in use"))
				return
			}
			log.Error("failed to delete document type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete document type"))
			return
		}

		log.Info("document type deleted", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Document Handlers ---

func GetDocuments(log *slog.Logger, repo DocumentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetDocuments"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.DocumentFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'employee_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'employee_id' parameter"))
				return
			}
			filter.EmployeeID = &val
		}

		if typeIDStr := q.Get("document_type_id"); typeIDStr != "" {
			val, err := strconv.Atoi(typeIDStr)
			if err != nil {
				log.Warn("invalid 'document_type_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'document_type_id' parameter"))
				return
			}
			filter.DocumentTypeID = &val
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if search := q.Get("search"); search != "" {
			filter.Search = &search
		}

		if expiringDaysStr := q.Get("expiring_days"); expiringDaysStr != "" {
			val, err := strconv.Atoi(expiringDaysStr)
			if err != nil {
				log.Warn("invalid 'expiring_days' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'expiring_days' parameter"))
				return
			}
			filter.ExpiringDays = &val
		}

		if expiredStr := q.Get("expired"); expiredStr != "" {
			val := expiredStr == "true"
			filter.Expired = &val
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, _ := strconv.Atoi(limitStr)
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, _ := strconv.Atoi(offsetStr)
			filter.Offset = val
		}

		documents, err := repo.GetDocuments(r.Context(), filter)
		if err != nil {
			log.Error("failed to get documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve documents"))
			return
		}

		log.Info("successfully retrieved documents", slog.Int("count", len(documents)))
		render.JSON(w, r, documents)
	}
}

func GetDocumentByID(log *slog.Logger, repo DocumentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetDocumentByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		document, err := repo.GetDocumentByID(r.Context(), id)
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

		render.JSON(w, r, document)
	}
}

func AddDocument(log *slog.Logger, repo DocumentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.AddDocument"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddDocumentRequest
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

		// Get creator ID from JWT claims
		var createdBy *int64
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if ok {
			createdBy = &claims.UserID
		}

		id, err := repo.AddDocument(r.Context(), req, createdBy)
		if err != nil {
			log.Error("failed to add document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add document"))
			return
		}

		log.Info("document added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func EditDocument(log *slog.Logger, repo DocumentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.EditDocument"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditDocumentRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditDocument(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("document not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document not found"))
				return
			}
			log.Error("failed to update document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update document"))
			return
		}

		log.Info("document updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteDocument(log *slog.Logger, repo DocumentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.DeleteDocument"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteDocument(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("document not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("document has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: document has signatures"))
				return
			}
			log.Error("failed to delete document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete document"))
			return
		}

		log.Info("document deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Signature Handlers ---

func GetSignatures(log *slog.Logger, repo SignatureRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetSignatures"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.SignatureFilter
		q := r.URL.Query()

		if docIDStr := q.Get("document_id"); docIDStr != "" {
			val, err := strconv.ParseInt(docIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'document_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'document_id' parameter"))
				return
			}
			filter.DocumentID = &val
		}

		if signerIDStr := q.Get("signer_user_id"); signerIDStr != "" {
			val, err := strconv.ParseInt(signerIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'signer_user_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'signer_user_id' parameter"))
				return
			}
			filter.SignerUserID = &val
		}

		if role := q.Get("signer_role"); role != "" {
			filter.SignerRole = &role
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		signatures, err := repo.GetHRMDocumentSignatures(r.Context(), filter)
		if err != nil {
			log.Error("failed to get signatures", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve signatures"))
			return
		}

		log.Info("successfully retrieved signatures", slog.Int("count", len(signatures)))
		render.JSON(w, r, signatures)
	}
}

func AddSignature(log *slog.Logger, repo SignatureRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.AddSignature"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddSignatureRequest
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

		id, err := repo.AddDocumentSignature(r.Context(), req)
		if err != nil {
			log.Error("failed to add signature", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add signature"))
			return
		}

		log.Info("signature added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// SignRequest represents a sign/reject request
type SignRequest struct {
	Signed          bool    `json:"signed"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

func SignDocument(log *slog.Logger, repo SignatureRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.SignDocument"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req SignRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Get client IP
		ip := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ip = forwardedFor
		}

		err = repo.SignHRMDocument(r.Context(), id, req.Signed, req.RejectionReason, ip)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("signature not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Signature not found"))
				return
			}
			log.Error("failed to process signature", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to process signature"))
			return
		}

		action := "signed"
		if !req.Signed {
			action = "rejected"
		}
		log.Info("document "+action, slog.Int64("signature_id", id))
		render.JSON(w, r, resp.OK())
	}
}

// --- Template Handlers ---

func GetTemplates(log *slog.Logger, repo TemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetTemplates"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.DocumentTemplateFilter
		q := r.URL.Query()

		if typeIDStr := q.Get("document_type_id"); typeIDStr != "" {
			val, err := strconv.Atoi(typeIDStr)
			if err != nil {
				log.Warn("invalid 'document_type_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'document_type_id' parameter"))
				return
			}
			filter.DocumentTypeID = &val
		}

		if isActiveStr := q.Get("is_active"); isActiveStr != "" {
			val := isActiveStr == "true"
			filter.IsActive = &val
		}

		if search := q.Get("search"); search != "" {
			filter.Search = &search
		}

		templates, err := repo.GetDocumentTemplates(r.Context(), filter)
		if err != nil {
			log.Error("failed to get templates", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve templates"))
			return
		}

		log.Info("successfully retrieved templates", slog.Int("count", len(templates)))
		render.JSON(w, r, templates)
	}
}

func GetTemplateByID(log *slog.Logger, repo TemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetTemplateByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		template, err := repo.GetDocumentTemplateByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("template not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Template not found"))
				return
			}
			log.Error("failed to get template", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve template"))
			return
		}

		render.JSON(w, r, template)
	}
}

func AddTemplate(log *slog.Logger, repo TemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.AddTemplate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddDocumentTemplateRequest
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

		// Get creator ID from JWT claims
		var createdBy *int64
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if ok {
			createdBy = &claims.UserID
		}

		id, err := repo.AddDocumentTemplate(r.Context(), req, createdBy)
		if err != nil {
			log.Error("failed to add template", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add template"))
			return
		}

		log.Info("template added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func EditTemplate(log *slog.Logger, repo TemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.EditTemplate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditDocumentTemplateRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditDocumentTemplate(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("template not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Template not found"))
				return
			}
			log.Error("failed to update template", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update template"))
			return
		}

		log.Info("template updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteTemplate(log *slog.Logger, repo TemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.DeleteTemplate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteDocumentTemplate(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("template not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Template not found"))
				return
			}
			log.Error("failed to delete template", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete template"))
			return
		}

		log.Info("template deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
