package past_events

import (
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/file"
	"time"
)

type EventType string

const (
	EventTypeInfo    EventType = "info"
	EventTypeWarning EventType = "warning"
	EventTypeDanger  EventType = "danger"
	EventTypeSuccess EventType = "success"
)

type Event struct {
	Type             EventType    `json:"type"`
	Date             time.Time    `json:"date"`
	OrganizationID   *int64       `json:"organization_id"`
	OrganizationName *string      `json:"organization_name"`
	Description      string       `json:"description"`
	EntityType       string       `json:"entity_type,omitempty"` // "incident", "shutdown", "discharge"
	EntityID         int64        `json:"entity_id"`
	Files            []file.Model `json:"files,omitempty"` // Files (will be transformed with presigned URLs in handler)
}

// EventWithURLs is the API response model with presigned file URLs
type EventWithURLs struct {
	Type             EventType          `json:"type"`
	Date             time.Time          `json:"date"`
	OrganizationID   *int64             `json:"organization_id"`
	OrganizationName *string            `json:"organization_name"`
	Description      string             `json:"description"`
	EntityType       string             `json:"entity_type,omitempty"` // "incident", "shutdown", "discharge"
	EntityID         int64              `json:"entity_id"`
	Files            []dto.FileResponse `json:"files,omitempty"` // Files with presigned URLs
}

type DateGroup struct {
	Date   string  `json:"date"`
	Events []Event `json:"events"`
}

// DateGroupWithURLs is the API response model with presigned file URLs
type DateGroupWithURLs struct {
	Date   string          `json:"date"`
	Events []EventWithURLs `json:"events"`
}

type Response struct {
	EventsByDate []DateGroup `json:"events_by_date"`
}

// ResponseWithURLs is the API response model with presigned file URLs
type ResponseWithURLs struct {
	EventsByDate []DateGroupWithURLs `json:"events_by_date"`
}

type Request struct {
	Days int `json:"days" validate:"omitempty,min=1,max=365"`
}
