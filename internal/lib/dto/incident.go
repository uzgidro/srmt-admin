package dto

import "time"

type EditIncidentRequest struct {
	OrganizationID *int64
	IncidentTime   *time.Time
	Description    *string
}
