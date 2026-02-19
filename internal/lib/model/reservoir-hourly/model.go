package reservoirhourly

import "time"

// BaseParams holds current and day-begin values
type BaseParams struct {
	Current  float64 `json:"current"`
	DayBegin float64 `json:"day_begin"`
}

// WeatherParams holds current and day-begin weather strings
type WeatherParams struct {
	Current  string `json:"current"`
	DayBegin string `json:"day_begin"`
}

// ReservoirData holds processed hourly data for one reservoir
type ReservoirData struct {
	OrganizationID   int64         `json:"organization_id"`
	OrganizationName string        `json:"organization_name"`
	Weather          WeatherParams `json:"weather"`
	Level            BaseParams    `json:"level"`
	Volume           BaseParams    `json:"volume"`
	Income           []float64     `json:"income"`
	Release          float64       `json:"release"`
	IncomeAtDayBegin float64       `json:"income_at_day_begin"`
}

// HourlyReport is the top-level report structure
type HourlyReport struct {
	Date       string          `json:"date"`
	LatestTime time.Time       `json:"latest_time"`
	Period     int             `json:"period"`
	Reservoirs []ReservoirData `json:"reservoirs"`
}
