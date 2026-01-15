package decree_type

// Model represents a decree type
type Model struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
