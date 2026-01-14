package legaldocuments

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	legal_document "srmt-admin/internal/lib/model/legal-document"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type documentGetter interface {
	GetAllLegalDocuments(ctx context.Context, filters dto.GetAllLegalDocumentsFilters) ([]*legal_document.ResponseModel, error)
}

func GetAll(log *slog.Logger, getter documentGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.legal-document.get-all"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse query parameters for filtering
		filters := dto.GetAllLegalDocumentsFilters{}
		q := r.URL.Query()

		// Filter by type_id
		if typeIDStr := q.Get("type_id"); typeIDStr != "" {
			typeID, err := strconv.Atoi(typeIDStr)
			if err != nil {
				log.Warn("invalid 'type_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'type_id' parameter"))
				return
			}
			filters.TypeID = &typeID
		}

		// Filter by start_date
		if startDateStr := q.Get("start_date"); startDateStr != "" {
			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				log.Warn("invalid 'start_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'start_date' parameter (use YYYY-MM-DD format)"))
				return
			}
			filters.StartDate = &startDate
		}

		// Filter by end_date
		if endDateStr := q.Get("end_date"); endDateStr != "" {
			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				log.Warn("invalid 'end_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'end_date' parameter (use YYYY-MM-DD format)"))
				return
			}
			filters.EndDate = &endDate
		}

		// Search by name
		if nameSearch := q.Get("name"); nameSearch != "" {
			filters.NameSearch = &nameSearch
		}

		// Search by number
		if numberSearch := q.Get("number"); numberSearch != "" {
			filters.NumberSearch = &numberSearch
		}

		documents, err := getter.GetAllLegalDocuments(r.Context(), filters)
		if err != nil {
			log.Error("failed to get all legal documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve legal documents"))
			return
		}

		// Transform documents to include presigned URLs
		documentsWithURLs := make([]*legal_document.ResponseWithURLs, 0, len(documents))
		for _, doc := range documents {
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
			documentsWithURLs = append(documentsWithURLs, docWithURLs)
		}

		log.Info("successfully retrieved legal documents", slog.Int("count", len(documentsWithURLs)))
		render.JSON(w, r, documentsWithURLs)
	}
}
