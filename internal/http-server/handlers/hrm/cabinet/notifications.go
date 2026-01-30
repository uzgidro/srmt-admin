package cabinet

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// NotificationRepository defines the interface for notification operations
type NotificationRepository interface {
	GetNotifications(ctx context.Context, filter hrm.NotificationFilter) ([]*hrmmodel.Notification, error)
	GetNotificationCount(ctx context.Context, userID int64) (*hrmmodel.NotificationCount, error)
	MarkNotificationRead(ctx context.Context, id int64, userID int64) error
	MarkAllNotificationsRead(ctx context.Context, userID int64, category *string) error
}

// GetMyNotifications returns notifications for the currently authenticated user
func GetMyNotifications(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMyNotifications"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Parse query params
		category := r.URL.Query().Get("category")
		isReadStr := r.URL.Query().Get("is_read")

		filter := hrm.NotificationFilter{
			UserID: claims.UserID,
			Limit:  50,
		}

		if category != "" {
			filter.Category = &category
		}

		if isReadStr != "" {
			isRead := isReadStr == "true"
			filter.IsRead = &isRead
		}

		// Get notifications
		notifications, err := repo.GetNotifications(r.Context(), filter)
		if err != nil {
			log.Error("failed to get notifications", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve notifications"))
			return
		}

		// Get notification count
		count, err := repo.GetNotificationCount(r.Context(), claims.UserID)
		if err != nil {
			log.Warn("failed to get notification count", sl.Err(err))
			// Non-fatal, continue without count
		}

		// Build response
		response := struct {
			Notifications []hrm.MyNotificationResponse `json:"notifications"`
			Count         *hrmmodel.NotificationCount  `json:"count,omitempty"`
		}{
			Notifications: make([]hrm.MyNotificationResponse, 0, len(notifications)),
			Count:         count,
		}

		for _, n := range notifications {
			response.Notifications = append(response.Notifications, hrm.MyNotificationResponse{
				ID:          n.ID,
				Title:       n.Title,
				Message:     n.Message,
				Category:    n.Category,
				Priority:    n.Priority,
				IsRead:      n.IsRead,
				ReadAt:      n.ReadAt,
				ActionURL:   n.ActionURL,
				ActionLabel: n.ActionLabel,
				CreatedAt:   n.CreatedAt,
			})
		}

		log.Info("notifications retrieved", slog.Int64("user_id", claims.UserID), slog.Int("count", len(notifications)))
		render.JSON(w, r, response)
	}
}

// MarkNotificationRead marks a notification as read
func MarkNotificationRead(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.MarkNotificationRead"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Get notification ID from URL
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid id parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// Mark as read (userID ensures ownership)
		if err := repo.MarkNotificationRead(r.Context(), id, claims.UserID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("notification not found or already read", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Notification not found"))
				return
			}
			log.Error("failed to mark notification read", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to mark notification as read"))
			return
		}

		log.Info("notification marked as read", slog.Int64("user_id", claims.UserID), slog.Int64("notification_id", id))
		render.JSON(w, r, resp.OK())
	}
}

// MarkAllNotificationsRead marks all notifications as read
func MarkAllNotificationsRead(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.MarkAllNotificationsRead"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Optional category filter
		var category *string
		categoryStr := r.URL.Query().Get("category")
		if categoryStr != "" {
			category = &categoryStr
		}

		// Mark all as read
		if err := repo.MarkAllNotificationsRead(r.Context(), claims.UserID, category); err != nil {
			log.Error("failed to mark all notifications read", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to mark notifications as read"))
			return
		}

		log.Info("all notifications marked as read", slog.Int64("user_id", claims.UserID))
		render.JSON(w, r, resp.OK())
	}
}
