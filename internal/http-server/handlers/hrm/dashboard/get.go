package dashboard

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/dashboard"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ServiceInterface interface {
	GetDashboard(ctx context.Context, userID int64) (*dashboard.Data, error)
}

func Get(log *slog.Logger, svc ServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.dashboard.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		data, err := svc.GetDashboard(r.Context(), claims.ContactID)
		if err != nil {
			log.Error("failed to get dashboard", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to load dashboard"))
			return
		}

		render.JSON(w, r, data)
	}
}
