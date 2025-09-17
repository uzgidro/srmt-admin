package complex_value

import "srmt-admin/internal/lib/model/dto/value"

type Model struct {
	ReservoirID int           `json:"reservoir_id"`
	Reservoir   string        `json:"reservoir"`
	AvgIncome   float64       `json:"avg_income"`
	Data        []value.Model `json:"data"`
}
