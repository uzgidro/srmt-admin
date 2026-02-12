package dashboard

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type AllNotificationReader interface {
	MarkAllNotificationsRead(ctx context.Context, userID int64) error
}

func MarkReadAll(log *slog.Logger, svc AllNotificationReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.dashboard.MarkReadAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		if err := svc.MarkAllNotificationsRead(r.Context(), claims.ContactID); err != nil {
			log.Error("failed to mark all notifications read", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to mark all notifications read"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
