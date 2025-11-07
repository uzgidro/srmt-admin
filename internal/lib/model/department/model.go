package department

import (
	"srmt-admin/internal/lib/model/organization"
	"time"
)

type Model struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	OrganizationID int64      `json:"organization_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`

	Organization *organization.Model `json:"organization,omitempty"`
}
