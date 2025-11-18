package user

import (
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/position"
	"time"
)

type Model struct {
	ID        int64  `json:"id"`        // User ID
	IsActive  bool   `json:"is_active"` // From users table
	Login     string `json:"login"`     // From users table
	ContactID int64  `json:"contact_id"`

	// --- Данные из Contacts ---
	Name            string     `json:"name"`
	Email           *string    `json:"email,omitempty"`
	Phone           *string    `json:"phone,omitempty"`
	IPPhone         *string    `json:"ip_phone,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	ExternalOrgName *string    `json:"external_organization_name,omitempty"`

	// --- Вложенные "обогащенные" модели ---
	Organization *organization.Model `json:"organization,omitempty"`
	Department   *department.Model   `json:"department,omitempty"`
	Position     *position.Model     `json:"position,omitempty"`

	// --- Роли (из твоего старого кода) ---
	Roles []string `json:"roles"`

	// --- Аудит (из users) ---
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
