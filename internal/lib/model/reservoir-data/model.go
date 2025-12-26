package reservoirdata

// ReservoirDataItem represents a single reservoir data record
type ReservoirDataItem struct {
	OrganizationID            int64    `json:"organization_id" validate:"required"`
	Date                      string   `json:"date" validate:"required"`
	Income                    float64  `json:"income"`
	Level                     float64  `json:"level"`
	Release                   float64  `json:"release"`
	Volume                    float64  `json:"volume"`
	ModsnowCurrent            *float64 `json:"modsnow_current"`
	ModsnowYearAgo            *float64 `json:"modsnow_year_ago"`
	TotalIncomeVolume         *float64 `json:"total_income_volume,omitempty"`
	TotalIncomeVolumePrevYear *float64 `json:"total_income_volume_prev_year,omitempty"`
}
