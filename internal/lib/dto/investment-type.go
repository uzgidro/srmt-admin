package dto

// AddInvestmentTypeRequest is the DTO for creating an investment type
type AddInvestmentTypeRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// EditInvestmentTypeRequest is the DTO for updating an investment type
type EditInvestmentTypeRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}
