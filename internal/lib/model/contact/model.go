package contact

import (
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"time"
)

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

	// Аудит
	CreatedAt time.Time  `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}
