package dto

import "time"

type AddShutdownRequest struct {
	OrganizationID      int64      `json:"organization_id" validate:"required"`
	StartTime           time.Time  `json:"start_time" validate:"required"`
	EndTime             *time.Time `json:"end_time,omitempty"`
	Reason              *string    `json:"reason,omitempty"`
	GenerationLossMwh   *float64   `json:"generation_loss,omitempty"`
	ReportedByContactID *int64     `json:"reported_by_contact_id,omitempty"`
	CreatedByUserID     int64      `json:"-"`

	IdleDischargeVolumeThousandM3 *float64 `json:"idle_discharge_volume,omitempty"`
	FileIDs                       []int64  `json:"file_ids,omitempty"`
}

type EditShutdownRequest struct {
	OrganizationID      *int64     `json:"organization_id,omitempty"`
	StartTime           *time.Time `json:"start_time,omitempty"`
	EndTime             *time.Time `json:"end_time,omitempty"`
	Reason              *string    `json:"reason,omitempty"`
	GenerationLossMwh   *float64   `json:"generation_loss,omitempty"`
	ReportedByContactID *int64     `json:"reported_by_contact_id,omitempty"`

	IdleDischargeVolumeThousandM3 *float64 `json:"idle_discharge_volume,omitempty"`

	// CreatedByUserID is used when creating a new idle discharge during edit
	CreatedByUserID int64   `json:"-"`
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type GetShutdownsFilters struct {
	Day time.Time
}
