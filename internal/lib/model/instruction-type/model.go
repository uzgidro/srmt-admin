package instruction_type

// Model represents an instruction type
type Model struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
