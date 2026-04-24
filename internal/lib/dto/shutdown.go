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
	Force                         bool
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

	// CreatedByUserID stamps a NEWLY created idle_water_discharges row produced
	// during this edit. It MUST NOT be written to shutdowns.created_by_user_id
	// — that column is set once at INSERT time and is the authority for the
	// cascade-owner restriction (auth.CheckShutdownOwnership). Letting an edit
	// rewrite it would let any caller transfer ownership to themselves and
	// bypass the check on the next request.
	CreatedByUserID int64
	FileIDs         []int64 `json:"file_ids,omitempty"`
}

type GetShutdownsFilters struct {
	Day time.Time
}
