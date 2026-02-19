package dto

import "time"

type AddIncidentRequest struct {
	OrganizationID  *int64    `json:"organization_id,omitempty"`
	IncidentTime    time.Time `json:"incident_time" validate:"required"`
	Description     string    `json:"description" validate:"required"`
	FileIDs         []int64   `json:"file_ids,omitempty"`
	CreatedByUserID int64     `json:"-"`
}

type EditIncidentRequest struct {
	OrganizationID *int64     `json:"organization_id,omitempty"`
	IncidentTime   *time.Time `json:"incident_time,omitempty"`
	Description    *string    `json:"description,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
}
