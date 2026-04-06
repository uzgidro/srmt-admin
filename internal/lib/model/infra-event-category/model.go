package infraeventcategory

import "time"

type Model struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name"`
	Label       string    `json:"label"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}
