package dto

import "time"

type GetAllContactsFilters struct {
	OrganizationID *int64
	DepartmentID   *int64
}

// AddContactRequest - DTO для создания контакта
type AddContactRequest struct {
	Name            string
	Email           *string
	Phone           *string
	IPPhone         *string
	DOB             *time.Time
	ExternalOrgName *string
	IconID          *int64
	OrganizationID  *int64 // Nullable
	DepartmentID    *int64 // Nullable
	PositionID      *int64 // Nullable
}

// EditContactRequest - DTO для обновления контакта
type EditContactRequest struct {
	Name            *string
	Email           *string
	Phone           *string
	IPPhone         *string
	DOB             *time.Time
	ExternalOrgName *string
	IconID          *int64
	OrganizationID  *int64
	DepartmentID    *int64
	PositionID      *int64
}
