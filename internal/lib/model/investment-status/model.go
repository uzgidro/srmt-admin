package investment_status

type Model struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	TypeID       *int   `json:"type_id,omitempty"` // NULL = shared status
	DisplayOrder int    `json:"display_order"`
}
