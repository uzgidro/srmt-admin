package notifications

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

type NotificationsGetter interface {
	GetHRMNotifications(ctx context.Context, userID int64) ([]*dashboard.Notification, error)
}

func GetAll(log *slog.Logger, repo NotificationsGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.notifications.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		notifications, err := repo.GetHRMNotifications(r.Context(), claims.ContactID)
		if err != nil {
			log.Error("failed to get notifications", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get notifications"))
			return
		}

		if notifications == nil {
			notifications = []*dashboard.Notification{}
		}
		render.JSON(w, r, notifications)
	}
}
