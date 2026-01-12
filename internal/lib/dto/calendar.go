package dto

// DayCounts represents the count of different event types for a single day
type DayCounts struct {
	Date       string `json:"date"`
	Incidents  int    `json:"incidents"`
	Shutdowns  int    `json:"shutdowns"`
	Discharges int    `json:"discharges"`
	Visits     int    `json:"visits"`
}

// CalendarResponse represents the calendar data for a month
type CalendarResponse struct {
	Year  int         `json:"year"`
	Month int         `json:"month"`
	Days  []DayCounts `json:"days"`
}
