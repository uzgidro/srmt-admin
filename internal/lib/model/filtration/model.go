package filtration

import "time"

// --- Filtration Location ---

type Location struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	Name           string    `json:"name"`
	Norm           *float64  `json:"norm"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateLocationRequest struct {
	OrganizationID int64    `json:"organization_id" validate:"required"`
	Name           string   `json:"name" validate:"required"`
	Norm           *float64 `json:"norm"`
	SortOrder      *int     `json:"sort_order"`
}

type UpdateLocationRequest struct {
	Name      *string  `json:"name"`
	Norm      *float64 `json:"norm"`
	SortOrder *int     `json:"sort_order"`
}

// --- Piezometer ---

type Piezometer struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	Name           string    `json:"name"`
	Norm           *float64  `json:"norm"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreatePiezometerRequest struct {
	OrganizationID int64    `json:"organization_id" validate:"required"`
	Name           string   `json:"name" validate:"required"`
	Norm           *float64 `json:"norm"`
	SortOrder      *int     `json:"sort_order"`
}

type UpdatePiezometerRequest struct {
	Name      *string  `json:"name"`
	Norm      *float64 `json:"norm"`
	SortOrder *int     `json:"sort_order"`
}

// --- Measurements ---

type FiltrationMeasurement struct {
	ID         int64    `json:"id"`
	LocationID int64    `json:"location_id"`
	Date       string   `json:"date"`
	FlowRate   *float64 `json:"flow_rate"`
}

type PiezometerMeasurement struct {
	ID           int64    `json:"id"`
	PiezometerID int64    `json:"piezometer_id"`
	Date         string   `json:"date"`
	Level        *float64 `json:"level"`
	Anomaly      bool     `json:"anomaly"`
}

type FiltrationMeasurementInput struct {
	LocationID int64    `json:"location_id" validate:"required"`
	FlowRate   *float64 `json:"flow_rate"`
}

type PiezometerMeasurementInput struct {
	PiezometerID int64    `json:"piezometer_id" validate:"required"`
	Level        *float64 `json:"level"`
	Anomaly      *bool    `json:"anomaly,omitempty"`
}

// --- Piezometer Counts (per organization) ---

type PiezometerCounts struct {
	Pressure    int `json:"pressure"`
	NonPressure int `json:"non_pressure"`
}

type PiezometerCountsRecord struct {
	ID               int64     `json:"id"`
	OrganizationID   int64     `json:"organization_id"`
	PressureCount    int       `json:"pressure_count"`
	NonPressureCount int       `json:"non_pressure_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UpsertPiezometerCountsRequest struct {
	OrganizationID   int64 `json:"organization_id" validate:"required"`
	PressureCount    int   `json:"pressure_count" validate:"gte=0"`
	NonPressureCount int   `json:"non_pressure_count" validate:"gte=0"`
}

// --- Aggregated response ---

type LocationReading struct {
	Location
	FlowRate *float64 `json:"flow_rate"`
}

type PiezoReading struct {
	Piezometer
	Level   *float64 `json:"level"`
	Anomaly bool     `json:"anomaly"`
}

type OrgFiltrationSummary struct {
	OrganizationID   int64             `json:"organization_id"`
	OrganizationName string            `json:"organization_name"`
	Locations        []LocationReading `json:"locations"`
	Piezometers      []PiezoReading    `json:"piezometers"`
	PiezoCounts      PiezometerCounts  `json:"piezometer_counts"`
}

// --- Comparison ---

type ComparisonSnapshot struct {
	Date        string            `json:"date"`
	Level       *float64          `json:"level"`
	Volume      *float64          `json:"volume"`
	Locations   []LocationReading `json:"locations"`
	Piezometers []PiezoReading    `json:"piezometers"`
	PiezoCounts PiezometerCounts  `json:"piezometer_counts"`
}

// --- Similar Dates & Comparison ---

type SimilarDate struct {
	Date   string   `json:"date"`
	Level  *float64 `json:"level"`
	Volume *float64 `json:"volume"`
}

type OrgSimilarDates struct {
	OrganizationID   int64         `json:"organization_id"`
	OrganizationName string        `json:"organization_name"`
	ReferenceDate    string        `json:"reference_date"`
	ReferenceLevel   *float64      `json:"reference_level"`
	ReferenceVolume  *float64      `json:"reference_volume"`
	SimilarDates     []SimilarDate `json:"similar_dates"`
}

type OrgComparisonV2 struct {
	OrganizationID   int64               `json:"organization_id"`
	OrganizationName string              `json:"organization_name"`
	Current          ComparisonSnapshot  `json:"current"`
	HistoricalFilter *ComparisonSnapshot `json:"historical_filter"`
	HistoricalPiezo  *ComparisonSnapshot `json:"historical_piezo"`
}
