package dto

import (
	"context"
	"srmt-admin/internal/lib/model/contact"
)

// ReservoirFetcher interface for fetching reservoir metrics
type ReservoirFetcher interface {
	FetchAll(ctx context.Context) (map[int64]*ReservoirMetrics, error)
	GetIDs() []int64
}

// ReservoirMetrics holds the current and differential reservoir data
type ReservoirMetrics struct {
	Current *ReservoirData `json:"current,omitempty"` // Current metrics from first element
	Diff    *ReservoirData `json:"diff,omitempty"`    // Difference between current and 6 o'clock reading
}

// ReservoirData represents reservoir water data at a specific point in time
type ReservoirData struct {
	Income  *float64 `json:"income,omitempty"`  // Water intake (приход)
	Release *float64 `json:"release,omitempty"` // Water release (расход)
	Level   *float64 `json:"level,omitempty"`   // Water level (уровень)
	Volume  *float64 `json:"volume,omitempty"`  // Water volume (объем)
}

// OrganizationWithReservoir represents an organization with reservoir metrics (flat structure)
type OrganizationWithReservoir struct {
	OrganizationID   int64             `json:"organization_id"`
	OrganizationName string            `json:"organization_name"`
	Contacts         []*contact.Model  `json:"contacts"`
	ReservoirMetrics *ReservoirMetrics `json:"reservoir_metrics,omitempty"`
}
