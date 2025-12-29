package dto

type GetAllInvestmentsFilters struct {
	StatusID        *int     `json:"status_id,omitempty"`
	MinCost         *float64 `json:"min_cost,omitempty"`
	MaxCost         *float64 `json:"max_cost,omitempty"`
	NameSearch      *string  `json:"name_search,omitempty"`
	CreatedByUserID *int64   `json:"created_by_user_id,omitempty"`
}

// AddInvestmentRequest is the DTO for creating an investment
type AddInvestmentRequest struct {
	Name     string  `json:"name"`
	StatusID int     `json:"status_id"`
	Cost     float64 `json:"cost"`
	Comments *string `json:"comments,omitempty"`
	FileIDs  []int64 `json:"file_ids,omitempty"`
}

// EditInvestmentRequest is the DTO for updating an investment
// All fields are pointers (optional) - only provided fields will be updated
type EditInvestmentRequest struct {
	Name     *string  `json:"name,omitempty"`
	StatusID *int     `json:"status_id,omitempty"`
	Cost     *float64 `json:"cost,omitempty"`
	Comments *string  `json:"comments,omitempty"`
	FileIDs  []int64  `json:"file_ids,omitempty"`
}
