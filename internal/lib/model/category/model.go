package category

type Model struct {
	ID int64 `json:"id"`
	// Указатель используется, чтобы можно было передать `nil`
	// для категорий верхнего уровня.
	ParentID    *int64 `json:"-"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
}

// GetID returns the category ID (implements CategoryModel interface)
func (m Model) GetID() int64 {
	return m.ID
}
