package document

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/document"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DocumentRequestAllGetter interface {
	GetAllRequests(ctx context.Context, employeeID *int64) ([]*document.DocumentRequest, error)
}

func GetRequests(log *slog.Logger, svc DocumentRequestAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.document.GetRequests"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		requests, err := svc.GetAllRequests(r.Context(), nil)
		if err != nil {
			log.Error("failed to get document requests", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve document requests"))
			return
		}

		render.JSON(w, r, requests)
	}
}
