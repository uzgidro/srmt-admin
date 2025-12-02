package fast_call

import (
	"srmt-admin/internal/lib/model/contact"
)

type Model struct {
	ID        int64 `json:"id"`
	ContactID int64 `json:"contact_id"`
	Position  int   `json:"position"`

	// Nested contact model
	Contact *contact.Model `json:"contact,omitempty"`
}
