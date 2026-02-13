package document

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/document"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DocumentAllGetter interface {
	GetAll(ctx context.Context, filters dto.HRDocumentFilters) ([]*document.HRDocument, error)
}

func GetDocuments(log *slog.Logger, svc DocumentAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetDocuments"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		var filters dto.HRDocumentFilters

		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("type"); v != "" {
			filters.Type = &v
		}
		if v := q.Get("category"); v != "" {
			filters.Category = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		docs, err := svc.GetAll(r.Context(), filters)
		if err != nil {
			log.Error("failed to get documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve documents"))
			return
		}

		render.JSON(w, r, docs)
	}
}
