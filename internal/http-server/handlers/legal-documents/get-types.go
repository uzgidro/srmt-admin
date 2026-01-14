package legaldocuments

import (
	"context"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	legal_document_type "srmt-admin/internal/lib/model/legal-document-type"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type documentTypeGetter interface {
	GetAllLegalDocumentTypes(ctx context.Context) ([]legal_document_type.Model, error)
}

func GetTypes(log *slog.Logger, getter documentTypeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.legal-document.get-types"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		types, err := getter.GetAllLegalDocumentTypes(r.Context())
		if err != nil {
			log.Error("failed to get legal document types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve legal document types"))
			return
		}

		log.Info("successfully retrieved legal document types", slog.Int("count", len(types)))
		render.JSON(w, r, types)
	}
}
