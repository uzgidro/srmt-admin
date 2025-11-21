package complex_value

import "srmt-admin/internal/lib/model/dto/value"

type Model struct {
	ReservoirID int           `json:"reservoir_id"`
	Reservoir   string        `json:"reservoir"`
	Data        []value.Model `json:"data"`
}
