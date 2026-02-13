package access

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/access"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AccessRequestAllGetter interface {
	GetAllRequests(ctx context.Context, employeeID *int64) ([]*access.AccessRequest, error)
}

func GetRequests(log *slog.Logger, svc AccessRequestAllGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetRequests"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		requests, err := svc.GetAllRequests(r.Context(), nil)
		if err != nil {
			log.Error("failed to get access requests", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access requests"))
			return
		}

		render.JSON(w, r, requests)
	}
}
