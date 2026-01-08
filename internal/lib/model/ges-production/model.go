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
