package category

type Model struct {
	ID int64 `json:"id"`
	// Указатель используется, чтобы можно было передать `nil`
	// для категорий верхнего уровня.
	ParentID    *int64 `json:"parent_id,omitempty"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
}
