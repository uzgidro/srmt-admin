package signatures

import (
	"context"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/signature"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type pendingDocumentsGetter interface {
	GetPendingSignatureDocuments(ctx context.Context) ([]signature.PendingDocument, error)
}

// GetPending returns all documents waiting for signature across all document types
func GetPending(log *slog.Logger, getter pendingDocumentsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.signatures.get-pending"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		documents, err := getter.GetPendingSignatureDocuments(r.Context())
		if err != nil {
			log.Error("failed to get pending signature documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve pending documents"))
			return
		}

		log.Info("successfully retrieved pending signature documents", slog.Int("count", len(documents)))
		render.JSON(w, r, documents)
	}
}
