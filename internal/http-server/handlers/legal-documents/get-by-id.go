package legaldocuments

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	legal_document "srmt-admin/internal/lib/model/legal-document"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type documentByIDGetter interface {
	GetLegalDocumentByID(ctx context.Context, id int64) (*legal_document.ResponseModel, error)
}

func GetByID(log *slog.Logger, getter documentByIDGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.legal-document.get-by-id"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		doc, err := getter.GetLegalDocumentByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("legal document not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Legal document not found"))
				return
			}
			log.Error("failed to get legal document", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve legal document"))
			return
		}

		// Transform to include presigned URLs
		docWithURLs := &legal_document.ResponseWithURLs{
			ID:           doc.ID,
			Name:         doc.Name,
			Number:       doc.Number,
			DocumentDate: doc.DocumentDate,
			Type:         doc.Type,
			CreatedAt:    doc.CreatedAt,
			CreatedBy:    doc.CreatedBy,
			UpdatedAt:    doc.UpdatedAt,
			UpdatedBy:    doc.UpdatedBy,
			Files:        helpers.TransformFilesWithURLs(r.Context(), doc.Files, minioRepo, log),
		}

		log.Info("successfully retrieved legal document", slog.Int64("id", id))
		render.JSON(w, r, docWithURLs)
	}
}
