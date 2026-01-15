package letter_type

// Model represents a letter type
type Model struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
