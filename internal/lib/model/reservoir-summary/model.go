package reservoirsummary

type ValueResponse struct {
	Current     float64 `json:"current"`
	Previous    float64 `json:"prev"`
	YearAgo     float64 `json:"year_ago"`
	TwoYearsAgo float64 `json:"two_years_ago"`
	IsEdited    *bool   `json:"is_edited,omitempty"`
}

type ResponseModel struct {
	OrganizationID   *int64 `json:"organization_id"`
	OrganizationName string `json:"organization_name"`

	Income  ValueResponse `json:"income"`
	Volume  ValueResponse `json:"volume"`
	Level   ValueResponse `json:"level"`
	Release ValueResponse `json:"release"`
	Modsnow ValueResponse `json:"modsnow"`

	IncomingVolume                     float64 `json:"incoming_volume"`
	IncomingVolumePrevYear             float64 `json:"incoming_volume_prev_year"`
	IncomingVolumeIsCalculated         bool    `json:"incoming_volume_is_calculated"`
	IncomingVolumePrevYearIsCalculated bool    `json:"incoming_volume_prev_year_is_calculated"`

	// Details about incremental calculation base (if used)
	IncomingVolumeBaseDate          *string  `json:"incoming_volume_base_date,omitempty"`
	IncomingVolumeBaseValue         *float64 `json:"incoming_volume_base_value,omitempty"`
	IncomingVolumePrevYearBaseDate  *string  `json:"incoming_volume_prev_year_base_date,omitempty"`
	IncomingVolumePrevYearBaseValue *float64 `json:"incoming_volume_prev_year_base_value,omitempty"`
}

// ReservoirSummaryConfig controls which organizations appear in the
// /reservoir-summary report and how they roll up into the ИТОГО row.
// Source of truth for both the JSON endpoint and the Excel exports.
//
// ModsnowEnabled gates the per-org modsnow value in both the Excel
// generator (cell left empty when false) and the JSON response
// (Modsnow.Current/YearAgo masked to 0). Default in the DB is TRUE so
// behaviour is opt-out per reservoir.
//
// VolumeSource governs how Volume.Current is resolved when the daily
// snapshot in reservoir_data is missing or stale: "static" (default) uses
// the snapshot first and only falls through to the level→volume curve and
// the static.uz fallback when it is zero; "level_volume" inverts that —
// the curve wins over the snapshot. See migration 000086 and the strategy
// switch in handlers/reservoir-summary/volume_compute.go.
type ReservoirSummaryConfig struct {
	ID               int64  `json:"id"`
	OrganizationID   int64  `json:"organization_id"`
	OrganizationName string `json:"organization_name,omitempty"`
	SortOrder        int    `json:"sort_order"`
	IncludeInTotal   bool   `json:"include_in_total"`
	ModsnowEnabled   bool   `json:"modsnow_enabled"`
	VolumeSource     string `json:"volume_source"`
}

// UpsertReservoirSummaryConfigRequest is the POST body for creating or
// updating one config row. Upsert key is organization_id (UNIQUE in DB).
//
// ModsnowEnabled is sent as a plain bool — the front-end is expected to
// always include it (default TRUE for new rows). If callers omit the
// JSON field, Go's zero-value (false) is what gets persisted, so the
// front-end is responsible for sending TRUE explicitly on first create.
//
// VolumeSource accepts "static" or "level_volume"; an empty string is
// rewritten to "static" by the handler before validation runs, so existing
// clients that never sent the field continue to work unchanged.
type UpsertReservoirSummaryConfigRequest struct {
	OrganizationID int64  `json:"organization_id" validate:"required,gt=0"`
	SortOrder      int    `json:"sort_order" validate:"gte=0"`
	IncludeInTotal bool   `json:"include_in_total"`
	ModsnowEnabled bool   `json:"modsnow_enabled"`
	// omitempty stays even though the handler rewrites "" to "static" before
	// validation — it's the safety net for a literal JSON `null`, which Go
	// decodes to "" and which we want treated identically to "missing field".
	VolumeSource string `json:"volume_source" validate:"omitempty,oneof=static level_volume"`
}
