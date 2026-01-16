package signatures

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/signature"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type signatureRejecter interface {
	RejectSignature(ctx context.Context, docType string, docID int64, reason *string, userID int64) error
	GetSignatureRejectedStatusInfo(ctx context.Context) (*dto.StatusInfo, error)
}

// Reject rejects a document signature with optional reason
func Reject(log *slog.Logger, rejecter signatureRejecter, docType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.signatures.reject"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("document_type", docType),
		)

		// Validate document type
		if !signature.IsValidDocumentType(docType) {
			log.Warn("invalid document type")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid document type"))
			return
		}

		// Get user ID from context
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		// Parse document ID
		idStr := chi.URLParam(r, "id")
		docID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// Decode request body
		var req dto.RejectSignatureRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Reject the signature
		err = rejecter.RejectSignature(r.Context(), docType, docID, req.Reason, userID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("document not found", slog.Int64("id", docID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Document not found"))
				return
			}
			// Check for "not in pending_signature status" error
			if errors.Is(err, storage.ErrInvalidStatus) ||
				(err != nil && containsStatusError(err.Error())) {
				log.Warn("document is not in pending_signature status", slog.Int64("id", docID))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Document is not in pending_signature status"))
				return
			}
			log.Error("failed to reject signature", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to reject signature"))
			return
		}

		// Get rejected status info for response
		rejectedStatus, err := rejecter.GetSignatureRejectedStatusInfo(r.Context())
		if err != nil {
			log.Warn("failed to get signature_rejected status info", sl.Err(err))
			// Still return success, just without status info
			render.JSON(w, r, dto.SignatureResponse{Status: "OK"})
			return
		}

		log.Info("document signature rejected",
			slog.String("document_type", docType),
			slog.Int64("document_id", docID),
			slog.Int64("rejected_by", userID),
		)

		render.JSON(w, r, dto.SignatureResponse{
			Status:    "OK",
			NewStatus: rejectedStatus,
		})
	}
}
