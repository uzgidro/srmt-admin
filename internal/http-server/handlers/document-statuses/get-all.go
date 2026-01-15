package documentstatuses

import (
	"context"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	document_status "srmt-admin/internal/lib/model/document-status"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type documentStatusGetter interface {
	GetAllDocumentStatuses(ctx context.Context) ([]document_status.Model, error)
}

func GetAll(log *slog.Logger, getter documentStatusGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.document-statuses.get-all"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		statuses, err := getter.GetAllDocumentStatuses(r.Context())
		if err != nil {
			log.Error("failed to get document statuses", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve document statuses"))
			return
		}

		log.Info("successfully retrieved document statuses", slog.Int("count", len(statuses)))
		render.JSON(w, r, statuses)
	}
}
