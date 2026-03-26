package manualcomparison

// --- Upsert input ---

type FilterInput struct {
	LocationID         int64    `json:"location_id" validate:"required"`
	FlowRate           *float64 `json:"flow_rate"`
	HistoricalFlowRate *float64 `json:"historical_flow_rate"`
}

type PiezoInput struct {
	PiezometerID    int64    `json:"piezometer_id" validate:"required"`
	Level           *float64 `json:"level"`
	Anomaly         *bool    `json:"anomaly,omitempty"`
	HistoricalLevel *float64 `json:"historical_level"`
}

type UpsertRequest struct {
	OrganizationID       int64
	Date                 string
	HistoricalFilterDate string
	HistoricalPiezoDate  string
	Filters              []FilterInput
	Piezos               []PiezoInput
	UserID               int64
}

// --- Read models ---

type FilterReading struct {
	LocationID         int64    `json:"location_id"`
	LocationName       string   `json:"location_name"`
	Norm               *float64 `json:"norm"`
	SortOrder          int      `json:"sort_order"`
	FlowRate           *float64 `json:"flow_rate"`
	HistoricalFlowRate *float64 `json:"historical_flow_rate"`
}

type PiezoReading struct {
	PiezometerID    int64    `json:"piezometer_id"`
	PiezometerName  string   `json:"piezometer_name"`
	Norm            *float64 `json:"norm"`
	SortOrder       int      `json:"sort_order"`
	Level           *float64 `json:"level"`
	Anomaly         bool     `json:"anomaly"`
	HistoricalLevel *float64 `json:"historical_level"`
}

type OrgManualComparison struct {
	OrganizationID       int64           `json:"organization_id"`
	OrganizationName     string          `json:"organization_name"`
	Date                 string          `json:"date"`
	HistoricalFilterDate string          `json:"historical_filter_date"`
	HistoricalPiezoDate  string          `json:"historical_piezo_date"`
	Filters              []FilterReading `json:"filters"`
	Piezometers          []PiezoReading  `json:"piezometers"`
}
