package complex_value

import "srmt-admin/internal/lib/model/dto/value"

type ComplexValue struct {
	ReservoirID int           `json:"reservoir_id"`
	Reservoir   string        `json:"reservoir"`
	AvgIncome   float64       `json:"avg_income"`
	Data        []value.Value `json:"data"`
}
