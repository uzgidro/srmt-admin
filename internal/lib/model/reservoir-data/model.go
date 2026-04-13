package reservoirdata

import (
	optional "srmt-admin/internal/lib/optional"
)

// ReservoirDataItem represents a single reservoir data record.
//
// The numeric fields (income/level/release/volume/total_income_volume*) use
// the three-state Optional wrapper so callers can do partial updates:
//   - field absent from JSON → existing DB value is preserved
//   - field is explicit null → writes NULL (all numeric columns are nullable)
//   - field is a number → writes that number (including 0)
type ReservoirDataItem struct {
	OrganizationID            int64                      `json:"organization_id" validate:"required"`
	Date                      string                     `json:"date" validate:"required"`
	Income                    optional.Optional[float64] `json:"income"`
	Level                     optional.Optional[float64] `json:"level"`
	Release                   optional.Optional[float64] `json:"release"`
	Volume                    optional.Optional[float64] `json:"volume"`
	ModsnowCurrent            *float64                   `json:"modsnow_current"`
	ModsnowYearAgo            *float64                   `json:"modsnow_year_ago"`
	TotalIncomeVolume         optional.Optional[float64] `json:"total_income_volume"`
	TotalIncomeVolumePrevYear optional.Optional[float64] `json:"total_income_volume_prev_year"`
}
