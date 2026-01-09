package dto

// AddInvestmentStatusRequest is the DTO for creating an investment status
type AddInvestmentStatusRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	TypeID       *int   `json:"type_id,omitempty"` // NULL = shared status for all types
	DisplayOrder int    `json:"display_order"`
}

// EditInvestmentStatusRequest is the DTO for updating an investment status
type EditInvestmentStatusRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	TypeID       *int    `json:"type_id,omitempty"` // NULL = shared status for all types
	DisplayOrder *int    `json:"display_order,omitempty"`
}
