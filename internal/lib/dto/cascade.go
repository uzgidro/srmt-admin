package dto

import (
	"context"
	"srmt-admin/internal/lib/model/contact"
)

// ASCUEFetcher interface for fetching ASCUE metrics
type ASCUEFetcher interface {
	FetchAll(ctx context.Context) (map[int64]*ASCUEMetrics, error)
}

// CascadeWithDetails represents a cascade organization with contacts and discharge information
type CascadeWithDetails struct {
	ID                     int64                 `json:"id"`
	Name                   string                `json:"name"`
	ParentOrganizationID   *int64                `json:"parent_organization_id,omitempty"`
	ParentOrganizationName *string               `json:"parent_organization,omitempty"`
	Types                  []string              `json:"types"`
	Contacts               []*contact.Model      `json:"contacts"`
	CurrentDischarge       *float64              `json:"current_discharge,omitempty"` // Текущий расход воды в м³/с
	ASCUEMetrics           *ASCUEMetrics         `json:"ascue_metrics,omitempty"`     // ASCUE metrics from external source
	Items                  []*CascadeWithDetails `json:"items,omitempty"`             // Nested organizations (HPPs for cascades)
}

// ASCUEMetrics holds the ASCUE (automated system for commercial electricity accounting) metrics
type ASCUEMetrics struct {
	Active          *float64 `json:"active,omitempty"`            // Active power (MW)
	Reactive        *float64 `json:"reactive,omitempty"`          // Reactive power (MVAr)
	PowerImport     *float64 `json:"power_import,omitempty"`      // Import power
	PowerExport     *float64 `json:"power_export,omitempty"`      // Export power
	OwnNeeds        *float64 `json:"own_needs,omitempty"`         // Own consumption
	Flow            *float64 `json:"flow,omitempty"`              // Water flow (m³/s)
	ActiveAggCount  *int     `json:"active_agg_count,omitempty"`  // Number of active aggregates
	PendingAggCount *int     `json:"pending_agg_count,omitempty"` // Number of pending aggregates
	RepairAggCount  *int     `json:"repair_agg_count,omitempty"`  // Number of aggregates under repair
}
