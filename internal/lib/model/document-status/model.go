package document_status

import "time"

// Model represents a document workflow status
type Model struct {
	ID           int    `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	DisplayOrder int    `json:"display_order"`
	IsTerminal   bool   `json:"is_terminal"`
}

// ShortModel is a minimal status representation
type ShortModel struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// HistoryEntry represents a status change history record
type HistoryEntry struct {
	ID          int64     `json:"id"`
	FromStatus  *Model    `json:"from_status,omitempty"`
	ToStatus    Model     `json:"to_status"`
	ChangedAt   time.Time `json:"changed_at"`
	ChangedByID *int64    `json:"changed_by_id,omitempty"`
	ChangedBy   *string   `json:"changed_by,omitempty"`
	Comment     *string   `json:"comment,omitempty"`
}
