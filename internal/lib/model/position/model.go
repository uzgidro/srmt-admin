package position

import "time"

type Model struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   *time.Time `json:"-"`
}
