package past_events

import "time"

type EventType string

const (
	EventTypeInfo    EventType = "info"
	EventTypeWarning EventType = "warning"
	EventTypeDanger  EventType = "danger"
	EventTypeSuccess EventType = "success"
)

type Event struct {
	Type             EventType `json:"type"`
	Date             time.Time `json:"date"`
	OrganizationID   *int64    `json:"organization_id"`
	OrganizationName *string   `json:"organization_name"`
	Description      string    `json:"description"`
}

type Response struct {
	EventsByDate map[string][]Event `json:"events_by_date"`
}

type Request struct {
	Days int `json:"days" validate:"omitempty,min=1,max=365"`
}
