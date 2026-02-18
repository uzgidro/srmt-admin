package dto

// AddOrganizationRequest is the DTO for adding an organization
type AddOrganizationRequest struct {
	Name                 string  `json:"name" validate:"required"`
	ParentOrganizationID *int64  `json:"parent_organization_id,omitempty"`
	TypeIDs              []int64 `json:"type_ids" validate:"required,min=1"`
}
