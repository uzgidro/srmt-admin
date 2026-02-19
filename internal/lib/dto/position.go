package dto

// AddPositionRequest - DTO for adding a position
type AddPositionRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
}

// EditPositionRequest - DTO for editing a position
type EditPositionRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Description *string `json:"description,omitempty"`
}
