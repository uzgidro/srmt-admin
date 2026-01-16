package dto

import "time"

type AddShutdownRequest struct {
	OrganizationID      int64
	StartTime           time.Time
	EndTime             *time.Time
	Reason              *string
	GenerationLossMwh   *float64
	ReportedByContactID *int64
	CreatedByUserID     int64

	IdleDischargeVolumeThousandM3 *float64
	FileIDs                       []int64 `json:"file_ids,omitempty"`
}

type EditShutdownRequest struct {
	OrganizationID      *int64
	StartTime           *time.Time
	EndTime             *time.Time
	Reason              *string
	GenerationLossMwh   *float64
	ReportedByContactID *int64

	IdleDischargeVolumeThousandM3 *float64

	// CreatedByUserID is used when creating a new idle discharge during edit
	CreatedByUserID int64
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type GetShutdownsFilters struct {
	Day time.Time
}
