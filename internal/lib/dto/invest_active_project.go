package dto

type AddInvestActiveProjectRequest struct {
	Category             string   `json:"category"`
	ProjectName          string   `json:"project_name"`
	ForeignPartner       *string  `json:"foreign_partner,omitempty"`
	ImplementationPeriod *string  `json:"implementation_period,omitempty"`
	CapacityMW           *float64 `json:"capacity_mw,omitempty"`
	ProductionMlnKWh     *float64 `json:"production_mln_kwh,omitempty"`
	CostMlnUSD           *float64 `json:"cost_mln_usd,omitempty"`
	StatusText           *string  `json:"status_text,omitempty"`
}

type EditInvestActiveProjectRequest struct {
	Category             *string  `json:"category,omitempty"`
	ProjectName          *string  `json:"project_name,omitempty"`
	ForeignPartner       *string  `json:"foreign_partner,omitempty"`
	ImplementationPeriod *string  `json:"implementation_period,omitempty"`
	CapacityMW           *float64 `json:"capacity_mw,omitempty"`
	ProductionMlnKWh     *float64 `json:"production_mln_kwh,omitempty"`
	CostMlnUSD           *float64 `json:"cost_mln_usd,omitempty"`
	StatusText           *string  `json:"status_text,omitempty"`
}
