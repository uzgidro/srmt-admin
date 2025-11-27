package event

import (
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/event_status"
	"srmt-admin/internal/lib/model/event_type"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/user"
	"time"
)

type Model struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Location    *string   `json:"location,omitempty"`
	EventDate   time.Time `json:"event_date"`

	// Foreign key IDs
	ResponsibleContactID int64  `json:"responsible_contact_id"`
	EventStatusID        int    `json:"event_status_id"`
	EventTypeID          int    `json:"event_type_id"`
	OrganizationID       *int64 `json:"organization_id,omitempty"`
	CreatedByID          int64  `json:"created_by_id"`

	// Nested enriched models (populated by joins)
	ResponsibleContact *contact.Model      `json:"responsible_contact,omitempty"`
	EventStatus        *event_status.Model `json:"event_status,omitempty"`
	EventType          *event_type.Model   `json:"event_type,omitempty"`
	Organization       *organization.Model `json:"organization,omitempty"`
	CreatedBy          *user.Model         `json:"created_by,omitempty"`
	UpdatedBy          *user.Model         `json:"updated_by,omitempty"`
	Files              []file.Model        `json:"files,omitempty"`

	// Audit
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	UpdatedByID *int64     `json:"updated_by_id,omitempty"`
}

// ModelWithURLs is the API response model with presigned file URLs
type ModelWithURLs struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Location    *string   `json:"location,omitempty"`
	EventDate   time.Time `json:"event_date"`

	// Foreign key IDs
	ResponsibleContactID int64  `json:"responsible_contact_id"`
	EventStatusID        int    `json:"event_status_id"`
	EventTypeID          int    `json:"event_type_id"`
	OrganizationID       *int64 `json:"organization_id,omitempty"`
	CreatedByID          int64  `json:"created_by_id"`

	// Nested enriched models (populated by joins)
	ResponsibleContact *contact.Model      `json:"responsible_contact,omitempty"`
	EventStatus        *event_status.Model `json:"event_status,omitempty"`
	EventType          *event_type.Model   `json:"event_type,omitempty"`
	Organization       *organization.Model `json:"organization,omitempty"`
	CreatedBy          *user.Model         `json:"created_by,omitempty"`
	UpdatedBy          *user.Model         `json:"updated_by,omitempty"`
	Files              []dto.FileResponse  `json:"files,omitempty"`

	// Audit
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	UpdatedByID *int64     `json:"updated_by_id,omitempty"`
}
