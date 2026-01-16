package signatures

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/signature"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type documentSignaturesGetter interface {
	GetDocumentSignatures(ctx context.Context, docType string, docID int64) ([]signature.Signature, error)
}

// GetSignatures returns all signatures for a specific document
func GetSignatures(log *slog.Logger, getter documentSignaturesGetter, docType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.signatures.get-signatures"
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

		// Parse document ID
		idStr := chi.URLParam(r, "id")
		docID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// Get signatures
		signatures, err := getter.GetDocumentSignatures(r.Context(), docType, docID)
		if err != nil {
			log.Error("failed to get document signatures", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve signatures"))
			return
		}

		log.Info("successfully retrieved document signatures",
			slog.String("document_type", docType),
			slog.Int64("document_id", docID),
			slog.Int("count", len(signatures)),
		)
		render.JSON(w, r, signatures)
	}
}
