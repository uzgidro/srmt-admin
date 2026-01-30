package hrm

import "time"

// --- Notification DTOs ---

// AddNotificationRequest represents request to create notification
type AddNotificationRequest struct {
	UserID      int64      `json:"user_id" validate:"required"`
	Title       string     `json:"title" validate:"required"`
	Message     string     `json:"message" validate:"required"`
	Category    string     `json:"category" validate:"required,oneof=vacation document training review task system"`
	EntityType  *string    `json:"entity_type,omitempty"`
	EntityID    *int64     `json:"entity_id,omitempty"`
	Priority    string     `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	ActionURL   *string    `json:"action_url,omitempty"`
	ActionLabel *string    `json:"action_label,omitempty"`
	SendEmail   bool       `json:"send_email"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// BulkNotificationRequest represents request to send notification to multiple users
type BulkNotificationRequest struct {
	UserIDs     []int64    `json:"user_ids" validate:"required,min=1"`
	Title       string     `json:"title" validate:"required"`
	Message     string     `json:"message" validate:"required"`
	Category    string     `json:"category" validate:"required"`
	EntityType  *string    `json:"entity_type,omitempty"`
	EntityID    *int64     `json:"entity_id,omitempty"`
	Priority    string     `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	ActionURL   *string    `json:"action_url,omitempty"`
	ActionLabel *string    `json:"action_label,omitempty"`
	SendEmail   bool       `json:"send_email"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// MarkReadRequest represents request to mark notifications as read
type MarkReadRequest struct {
	NotificationIDs []int64 `json:"notification_ids" validate:"required,min=1"`
}

// MarkAllReadRequest represents request to mark all notifications as read
type MarkAllReadRequest struct {
	Category *string `json:"category,omitempty"` // Optional: only mark specific category
}

// NotificationFilter represents filter for notifications
type NotificationFilter struct {
	UserID   int64   `json:"user_id" validate:"required"`
	Category *string `json:"category,omitempty"`
	IsRead   *bool   `json:"is_read,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Limit    int     `json:"limit,omitempty"`
	Offset   int     `json:"offset,omitempty"`
}

// NotificationPreferences represents user notification preferences
type NotificationPreferences struct {
	UserID        int64 `json:"user_id"`
	EmailVacation bool  `json:"email_vacation"`
	EmailDocument bool  `json:"email_document"`
	EmailTraining bool  `json:"email_training"`
	EmailReview   bool  `json:"email_review"`
	EmailTask     bool  `json:"email_task"`
	EmailSystem   bool  `json:"email_system"`
	InAppEnabled  bool  `json:"in_app_enabled"`
}

// UpdateNotificationPreferencesRequest represents request to update preferences
type UpdateNotificationPreferencesRequest struct {
	EmailVacation *bool `json:"email_vacation,omitempty"`
	EmailDocument *bool `json:"email_document,omitempty"`
	EmailTraining *bool `json:"email_training,omitempty"`
	EmailReview   *bool `json:"email_review,omitempty"`
	EmailTask     *bool `json:"email_task,omitempty"`
	EmailSystem   *bool `json:"email_system,omitempty"`
	InAppEnabled  *bool `json:"in_app_enabled,omitempty"`
}
