package notification

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Repository Interfaces ---

type NotificationRepository interface {
	AddNotification(ctx context.Context, req hrm.AddNotificationRequest) (int64, error)
	GetNotificationByID(ctx context.Context, id int64) (*hrmmodel.Notification, error)
	GetNotifications(ctx context.Context, filter hrm.NotificationFilter) ([]*hrmmodel.Notification, error)
	CountUnreadNotifications(ctx context.Context, userID int64) (int, error)
	MarkNotificationRead(ctx context.Context, id int64, userID int64) error
	MarkAllNotificationsRead(ctx context.Context, userID int64, category *string) error
	DeleteNotification(ctx context.Context, id int64, userID int64) error
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// CountResponse represents a count response
type CountResponse struct {
	resp.Response
	Count int `json:"count"`
}

// --- Notification Handlers ---

func GetNotifications(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.notification.GetNotifications"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.NotificationFilter
		q := r.URL.Query()

		if userIDStr := q.Get("user_id"); userIDStr != "" {
			val, err := strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'user_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'user_id' parameter"))
				return
			}
			filter.UserID = val
		} else {
			// If no user_id provided, get current user's notifications
			claims, ok := mwauth.ClaimsFromContext(r.Context())
			if ok {
				filter.UserID = claims.UserID
			}
		}

		if category := q.Get("category"); category != "" {
			filter.Category = &category
		}

		if isReadStr := q.Get("is_read"); isReadStr != "" {
			val := isReadStr == "true"
			filter.IsRead = &val
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, _ := strconv.Atoi(limitStr)
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, _ := strconv.Atoi(offsetStr)
			filter.Offset = val
		}

		notifications, err := repo.GetNotifications(r.Context(), filter)
		if err != nil {
			log.Error("failed to get notifications", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve notifications"))
			return
		}

		log.Info("successfully retrieved notifications", slog.Int("count", len(notifications)))
		render.JSON(w, r, notifications)
	}
}

func GetNotificationByID(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.notification.GetNotificationByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		notification, err := repo.GetNotificationByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("notification not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Notification not found"))
				return
			}
			log.Error("failed to get notification", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve notification"))
			return
		}

		render.JSON(w, r, notification)
	}
}

func GetUnreadCount(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.notification.GetUnreadCount"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		count, err := repo.CountUnreadNotifications(r.Context(), claims.UserID)
		if err != nil {
			log.Error("failed to count unread notifications", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to count notifications"))
			return
		}

		log.Info("counted unread notifications", slog.Int("count", count))
		render.JSON(w, r, CountResponse{Response: resp.OK(), Count: count})
	}
}

func AddNotification(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.notification.AddNotification"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddNotificationRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddNotification(r.Context(), req)
		if err != nil {
			log.Error("failed to add notification", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add notification"))
			return
		}

		log.Info("notification added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func MarkAsRead(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.notification.MarkAsRead"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.MarkNotificationRead(r.Context(), id, claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("notification not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Notification not found"))
				return
			}
			log.Error("failed to mark notification as read", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update notification"))
			return
		}

		log.Info("notification marked as read", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func MarkAllAsRead(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.notification.MarkAllAsRead"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Optional category filter from query params
		var category *string
		if cat := r.URL.Query().Get("category"); cat != "" {
			category = &cat
		}

		err := repo.MarkAllNotificationsRead(r.Context(), claims.UserID, category)
		if err != nil {
			log.Error("failed to mark all notifications as read", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update notifications"))
			return
		}

		log.Info("all notifications marked as read", slog.Int64("user_id", claims.UserID))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteNotification(log *slog.Logger, repo NotificationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.notification.DeleteNotification"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteNotification(r.Context(), id, claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("notification not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Notification not found"))
				return
			}
			log.Error("failed to delete notification", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete notification"))
			return
		}

		log.Info("notification deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
