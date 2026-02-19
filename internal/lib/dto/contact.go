package dto

import "time"

type GetAllContactsFilters struct {
	OrganizationID *int64
	DepartmentID   *int64
}

// AddContactRequest - DTO для создания контакта
// AddContactRequest - DTO для создания контакта
type AddContactRequest struct {
	Name            string     `json:"name" validate:"required"`
	Email           *string    `json:"email,omitempty" validate:"omitempty,email"`
	Phone           *string    `json:"phone,omitempty"`
	IPPhone         *string    `json:"ip_phone,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	ExternalOrgName *string    `json:"external_organization_name,omitempty"`
	IconID          *int64     `json:"icon_id,omitempty"`
	OrganizationID  *int64     `json:"organization_id,omitempty"` // Nullable
	DepartmentID    *int64     `json:"department_id,omitempty"`   // Nullable
	PositionID      *int64     `json:"position_id,omitempty"`     // Nullable
}

// EditContactRequest - DTO для обновления контакта
type EditContactRequest struct {
	Name            *string    `json:"name,omitempty" validate:"omitempty,min=1"`
	Email           *string    `json:"email,omitempty" validate:"omitempty,email"`
	Phone           *string    `json:"phone,omitempty"`
	IPPhone         *string    `json:"ip_phone,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	ExternalOrgName *string    `json:"external_organization_name,omitempty"`
	IconID          *int64     `json:"icon_id,omitempty"`
	OrganizationID  *int64     `json:"organization_id,omitempty"`
	DepartmentID    *int64     `json:"department_id,omitempty"`
	PositionID      *int64     `json:"position_id,omitempty"`
}
