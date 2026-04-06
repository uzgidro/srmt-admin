package dto

import "time"

type AddInfraEventRequest struct {
	CategoryID      int64
	OrganizationID  int64
	OccurredAt      time.Time
	RestoredAt      *time.Time
	Description     string
	Remediation     *string
	Notes           *string
	CreatedByUserID int64
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type EditInfraEventRequest struct {
	CategoryID      *int64
	OrganizationID  *int64
	OccurredAt      *time.Time
	RestoredAt      *time.Time
	ClearRestoredAt bool // set restored_at to NULL (re-open event)
	Description     *string
	Remediation     *string
	Notes           *string
	FileIDs         []int64 `json:"file_ids,omitempty"`
}
