package hrm

import "time"

// Notification represents a user notification
type Notification struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"user_id"`

	// Content
	Title    string `json:"title"`
	Message  string `json:"message"`
	Category string `json:"category"`

	// Related entity
	EntityType *string `json:"entity_type,omitempty"`
	EntityID   *int64  `json:"entity_id,omitempty"`

	// Priority
	Priority string `json:"priority"`

	// Status
	IsRead bool       `json:"is_read"`
	ReadAt *time.Time `json:"read_at,omitempty"`

	// Action
	ActionURL   *string `json:"action_url,omitempty"`
	ActionLabel *string `json:"action_label,omitempty"`

	// Delivery
	SendEmail   bool       `json:"send_email"`
	EmailSentAt *time.Time `json:"email_sent_at,omitempty"`

	// Expiry
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// NotificationCategory constants
const (
	NotificationCategoryVacation = "vacation"
	NotificationCategoryDocument = "document"
	NotificationCategoryTraining = "training"
	NotificationCategoryReview   = "review"
	NotificationCategoryTask     = "task"
	NotificationCategorySystem   = "system"
)

// NotificationPriority constants
const (
	NotificationPriorityLow    = "low"
	NotificationPriorityNormal = "normal"
	NotificationPriorityHigh   = "high"
	NotificationPriorityUrgent = "urgent"
)

// NotificationEntityType constants
const (
	NotificationEntityVacation    = "vacation"
	NotificationEntityDocument    = "document"
	NotificationEntityTraining    = "training"
	NotificationEntityReview      = "review"
	NotificationEntityGoal        = "goal"
	NotificationEntityAssessment  = "assessment"
	NotificationEntityTimesheet   = "timesheet"
	NotificationEntityCertificate = "certificate"
	NotificationEntityInterview   = "interview"
)

// NotificationFilter represents filter for notifications
type NotificationFilter struct {
	UserID   int64   `json:"user_id"`
	Category *string `json:"category,omitempty"`
	IsRead   *bool   `json:"is_read,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Limit    int     `json:"limit,omitempty"`
	Offset   int     `json:"offset,omitempty"`
}

// NotificationCount represents notification count by read status
type NotificationCount struct {
	Total  int `json:"total"`
	Unread int `json:"unread"`
	Read   int `json:"read"`
}

// CreateNotificationParams represents parameters for creating notification
type CreateNotificationParams struct {
	UserID      int64
	Title       string
	Message     string
	Category    string
	Priority    string
	EntityType  *string
	EntityID    *int64
	ActionURL   *string
	ActionLabel *string
	SendEmail   bool
	ExpiresAt   *time.Time
}
