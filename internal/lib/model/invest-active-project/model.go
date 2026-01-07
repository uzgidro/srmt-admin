package invest_active_project

import "time"

type Model struct {
	ID                   int64     `json:"id"`
	Category             string    `json:"category"`
	ProjectName          string    `json:"project_name"`
	ForeignPartner       *string   `json:"foreign_partner,omitempty"`
	ImplementationPeriod *string   `json:"implementation_period,omitempty"`
	CapacityMW           *float64  `json:"capacity_mw,omitempty"`
	ProductionMlnKWh     *float64  `json:"production_mln_kwh,omitempty"`
	CostMlnUSD           *float64  `json:"cost_mln_usd,omitempty"`
	StatusText           *string   `json:"status_text,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}
