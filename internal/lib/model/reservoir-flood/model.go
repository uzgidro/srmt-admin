// Package reservoirflood provides domain models for the reservoir flood
// hourly observations and per-organization reporting configuration.
//
// Note: validator tags `gte=0` cannot be applied to optional.Optional[float64]
// directly because Optional unwraps to *float64 internally. Negative-value
// validation is performed in the handler (manual check on each Optional.Value
// when Set && Value != nil && *Value < 0). Validate tags on Optional fields
// are therefore only "omitempty".
package reservoirflood

import (
	"time"

	"srmt-admin/internal/lib/optional"
)

// Config is a per-organization toggle controlling whether the org appears in
// the reservoir-flood hourly reports.
type Config struct {
	ID               int64     `json:"id"`
	OrganizationID   int64     `json:"organization_id"`
	OrganizationName string    `json:"organization_name,omitempty"`
	SortOrder        int       `json:"sort_order"`
	IsActive         bool      `json:"is_active"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UpsertConfigRequest is the payload for POST/PUT config endpoints.
type UpsertConfigRequest struct {
	OrganizationID int64 `json:"organization_id" validate:"required"`
	SortOrder      int   `json:"sort_order"      validate:"gte=0"`
	IsActive       bool  `json:"is_active"`
}

// HourlyRecord represents a single hourly reservoir-flood observation row.
type HourlyRecord struct {
	ID               int64     `json:"id"`
	OrganizationID   int64     `json:"organization_id"`
	OrganizationName string    `json:"organization_name,omitempty"`
	RecordedAt       time.Time `json:"recorded_at"`
	WaterLevelM      *float64  `json:"water_level_m"`
	WaterVolumeMlnM3 *float64  `json:"water_volume_mln_m3"`
	InflowM3s        *float64  `json:"inflow_m3s"`
	OutflowM3s       *float64  `json:"outflow_m3s"`
	GESFlowM3s       *float64  `json:"ges_flow_m3s"`
	IdleDischargeM3s *float64  `json:"idle_discharge_m3s"`
	DutyName         *string   `json:"duty_name"`
	CapacityMwt      *float64  `json:"capacity_mwt"`
	WeatherCondition *string   `json:"weather_condition"`
	TemperatureC     *float64  `json:"temperature_c"`
	CreatedByUserID  *int64    `json:"created_by_user_id,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UpsertHourlyRequest is one item in the bulk-upsert request body.
//
// IMPORTANT: RecordedAt is a STRING in the request (handler parses it),
// not a time.Time. The handler normalizes it to the hour and re-marshals
// the normalized value back into RecordedAt before passing the slice to the
// repo. The captureRepo test mock asserts that the string value is
// "2026-04-27T15:00:00Z" after normalization.
type UpsertHourlyRequest struct {
	OrganizationID   int64                      `json:"organization_id" validate:"required"`
	RecordedAt       string                     `json:"recorded_at"     validate:"required"`
	WaterLevelM      optional.Optional[float64] `json:"water_level_m"       validate:"omitempty"`
	WaterVolumeMlnM3 optional.Optional[float64] `json:"water_volume_mln_m3" validate:"omitempty"`
	InflowM3s        optional.Optional[float64] `json:"inflow_m3s"          validate:"omitempty"`
	OutflowM3s       optional.Optional[float64] `json:"outflow_m3s"         validate:"omitempty"`
	GESFlowM3s       optional.Optional[float64] `json:"ges_flow_m3s"        validate:"omitempty"`
	IdleDischargeM3s optional.Optional[float64] `json:"idle_discharge_m3s"  validate:"omitempty"`
	DutyName         optional.Optional[string]  `json:"duty_name"`
	CapacityMwt      optional.Optional[float64] `json:"capacity_mwt"        validate:"omitempty"`
	WeatherCondition optional.Optional[string]  `json:"weather_condition"`
	TemperatureC     optional.Optional[float64] `json:"temperature_c"       validate:"omitempty"`
}
