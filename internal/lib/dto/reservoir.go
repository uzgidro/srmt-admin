package dto

import (
	"context"
	"srmt-admin/internal/lib/model/contact"
	"time"
)

// ReservoirFetcher interface for fetching reservoir metrics
type ReservoirFetcher interface {
	FetchAll(ctx context.Context, date string) (map[int64]*ReservoirMetrics, error)
	GetIDs() []int64
}

// ReservoirMetrics holds the current and differential reservoir data
type ReservoirMetrics struct {
	Current *ReservoirData `json:"current,omitempty"` // Current metrics from first element
	Diff    *ReservoirData `json:"diff,omitempty"`    // Difference between current and 6 o'clock reading
}

// ReservoirData represents reservoir water data at a specific point in time
type ReservoirData struct {
	ReservoirAPIID *int64     `json:"reservoir_api_id,omitempty"`
	Income         *float64   `json:"income,omitempty"`  // Water intake (приход)
	Release        *float64   `json:"release,omitempty"` // Water release (расход)
	Level          *float64   `json:"level,omitempty"`   // Water level (уровень)
	Volume         *float64   `json:"volume,omitempty"`  // Water volume (объем)
	Time           *time.Time `json:"time,omitempty"`    // Timestamp (combined date+time)
	Weather        *string    `json:"weather,omitempty"` // Weather condition
}

// OrganizationWithReservoir represents an organization with reservoir metrics (flat structure)
type OrganizationWithReservoir struct {
	OrganizationID   int64             `json:"organization_id"`
	OrganizationName string            `json:"organization_name"`
	Contacts         []*contact.Model  `json:"contacts"`
	CurrentDischarge float64           `json:"current_discharge"` // Current water discharge in m³/s (ongoing or end_date > now), 0 if none
	ReservoirMetrics *ReservoirMetrics `json:"reservoir_metrics,omitempty"`
}

type OrganizationWithData struct {
	OrganizationID int64          `json:"organization_id"`
	ReservoirAPIID int64          `json:"reservoir_api_id"`
	Data           *ReservoirData `json:"data,omitempty"`
}
