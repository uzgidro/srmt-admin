package dto

// AddRoleRequest - DTO for adding a role
type AddRoleRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
}

// EditRoleRequest - DTO for editing a role
type EditRoleRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Description *string `json:"description,omitempty"`
}
