package gesproduction

import "time"

type Model struct {
	ID                    int64     `json:"id"`
	Date                  string    `json:"date"` // YYYY-MM-DD
	TotalEnergyProduction float64   `json:"total_energy_production"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type DashboardResponse struct {
	Date            string  `json:"date"`
	Value           float64 `json:"value"`
	ChangePercent   float64 `json:"change_percent"`
	ChangeDirection string  `json:"change_direction"` // "up", "down", "flat"
}

type StatsResponse struct {
	Current    *CurrentValue `json:"current"`
	MonthTotal float64       `json:"month_total"`
	YearTotal  float64       `json:"year_total"`
}

type CurrentValue struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}
