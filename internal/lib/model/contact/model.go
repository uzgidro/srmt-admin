package contact

import (
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"time"
)

// IconFile represents the icon file information
type IconFile struct {
	ID        int64  `json:"id"`
	FileName  string `json:"file_name"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
}

type Model struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Email           *string    `json:"email,omitempty"`
	Phone           *string    `json:"phone,omitempty"`
	IPPhone         *string    `json:"ip_phone,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	ExternalOrgName *string    `json:"external_organization_name,omitempty"`
	IconID          *int64     `json:"icon_id,omitempty"`

	// Вложенные "обогащенные" модели
	Organization *organization.Model `json:"organization,omitempty"`
	Department   *department.Model   `json:"department,omitempty"`
	Position     *position.Model     `json:"position,omitempty"`
	Icon         *IconFile           `json:"icon,omitempty"`

	// Аудит
	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}
