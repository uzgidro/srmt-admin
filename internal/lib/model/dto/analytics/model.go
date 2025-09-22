package analytics

import "srmt-admin/internal/lib/model/dto/value"

type Model struct {
	ReservoirID int           `json:"reservoir_id"`
	Reservoir   string        `json:"reservoir"`
	Years       []value.Model `json:"years"`
	CurrentYear []value.Model `json:"current_year"`
	PastYear    []value.Model `json:"past_year"`
	Min         []value.Model `json:"min"`
	Max         []value.Model `json:"max"`
	Avg         []value.Model `json:"avg"`
	TenAvg      []value.Model `json:"ten_avg"`
}
