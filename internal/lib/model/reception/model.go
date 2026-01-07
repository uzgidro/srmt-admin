package reception

import (
	"srmt-admin/internal/lib/model/user"
	"time"
)

type Model struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Date               time.Time `json:"date"`
	Description        *string   `json:"description,omitempty"`
	Visitor            string    `json:"visitor"`
	Status             string    `json:"status"` // "default", "true", "false"
	StatusChangeReason *string   `json:"status_change_reason,omitempty"`
	Informed           bool      `json:"informed"`
	InformedByUserID   *int64    `json:"informed_by_user_id,omitempty"`

	// Audit fields
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	CreatedByID int64      `json:"created_by_id"`
	UpdatedByID *int64     `json:"updated_by_id,omitempty"`

	// Nested user models for audit info
	CreatedBy  *user.Model `json:"created_by,omitempty"`
	UpdatedBy  *user.Model `json:"updated_by,omitempty"`
	InformedBy *user.Model `json:"informed_by,omitempty"`
}
