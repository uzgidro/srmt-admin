package dto

// AddDepartmentRequest is the DTO for adding a department
type AddDepartmentRequest struct {
	Name           string  `json:"name" validate:"required"`
	Description    *string `json:"description,omitempty"`
	OrganizationID int64   `json:"organization_id" validate:"required"`
}

// EditDepartmentRequest is the DTO for editing a department
type EditDepartmentRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Description *string `json:"description,omitempty"`
}
