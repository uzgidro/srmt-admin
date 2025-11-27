package dto

import "time"

type AddIncidentRequest struct {
	OrganizationID *int64    `json:"organization_id,omitempty"`
	IncidentTime   time.Time `json:"incident_time"`
	Description    string    `json:"description"`
	FileIDs        []int64   `json:"file_ids,omitempty"`
}

type EditIncidentRequest struct {
	OrganizationID *int64
	IncidentTime   *time.Time
	Description    *string
	FileIDs        []int64 `json:"file_ids,omitempty"`
}
