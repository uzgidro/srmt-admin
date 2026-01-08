package reservoirdata

import "encoding/json"

// Optional represents a field that can be:
// 1. Absent (Set = false)
// 2. Null (Set = true, Value = nil)
// 3. Value (Set = true, Value = *T)
type Optional[T any] struct {
	Value *T
	Set   bool
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// ReservoirDataItem represents a single reservoir data record
type ReservoirDataItem struct {
	OrganizationID            int64             `json:"organization_id" validate:"required"`
	Date                      string            `json:"date" validate:"required"`
	Income                    float64           `json:"income"`
	Level                     float64           `json:"level"`
	Release                   float64           `json:"release"`
	Volume                    float64           `json:"volume"`
	ModsnowCurrent            *float64          `json:"modsnow_current"`
	ModsnowYearAgo            *float64          `json:"modsnow_year_ago"`
	TotalIncomeVolume         Optional[float64] `json:"total_income_volume"`
	TotalIncomeVolumePrevYear Optional[float64] `json:"total_income_volume_prev_year"`
}
