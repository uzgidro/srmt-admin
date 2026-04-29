// Package solar provides domain models for solar-panel stations:
// per-organization config, daily generation/grid-export readings, and
// monthly production plans.
//
// Note: validator tags `gte=0` cannot be applied to optional.Optional[float64]
// directly because Optional unwraps to *float64 internally. Negative-value
// validation is performed in the handler (manual check on each Optional.Value
// when Set && Value != nil && *Value < 0). Validate tags on Optional fields
// are therefore only "omitempty".
package solar

import (
	"time"

	"srmt-admin/internal/lib/optional"
)

// Config is per-organization solar configuration. One row per org with panels.
type Config struct {
	ID                  int64     `json:"id"`
	OrganizationID      int64     `json:"organization_id"`
	OrganizationName    string    `json:"organization_name,omitempty"`
	InstalledCapacityKW float64   `json:"installed_capacity_kw"`
	SortOrder           int       `json:"sort_order"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// UpsertConfigRequest is the payload for POST /solar/config.
type UpsertConfigRequest struct {
	OrganizationID      int64   `json:"organization_id" validate:"required"`
	InstalledCapacityKW float64 `json:"installed_capacity_kw" validate:"gte=0"`
	SortOrder           int     `json:"sort_order" validate:"gte=0"`
}

// DailyData is one persisted solar daily-data row enriched with the
// organization name for read endpoints.
type DailyData struct {
	ID               int64     `json:"id"`
	OrganizationID   int64     `json:"organization_id"`
	OrganizationName string    `json:"organization_name,omitempty"`
	Date             time.Time `json:"date"`
	GenerationKWh    *float64  `json:"generation_kwh"`
	GridExportKWh    *float64  `json:"grid_export_kwh"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UpsertDailyDataRequest is one item of the bulk POST /solar/daily-data
// payload. Date is a YYYY-MM-DD string parsed in the repo before write.
type UpsertDailyDataRequest struct {
	OrganizationID int64                      `json:"organization_id" validate:"required"`
	Date           string                     `json:"date" validate:"required"`
	GenerationKWh  optional.Optional[float64] `json:"generation_kwh"  validate:"omitempty"`
	GridExportKWh  optional.Optional[float64] `json:"grid_export_kwh" validate:"omitempty"`
}

// ProductionPlan is one persisted monthly plan row.
// PlanThousandKWh is in THOUSANDS of kWh (NOT mln kWh — solar is much smaller).
type ProductionPlan struct {
	ID               int64   `json:"id"`
	OrganizationID   int64   `json:"organization_id"`
	OrganizationName string  `json:"organization_name,omitempty"`
	Year             int     `json:"year"`
	Month            int     `json:"month"`
	PlanThousandKWh  float64 `json:"plan_thousand_kwh"`
}

// UpsertPlanRequest is one item of the bulk POST /solar/plans payload.
type UpsertPlanRequest struct {
	OrganizationID  int64   `json:"organization_id" validate:"required"`
	Year            int     `json:"year" validate:"required,gte=2020,lte=2100"`
	Month           int     `json:"month" validate:"required,gte=1,lte=12"`
	PlanThousandKWh float64 `json:"plan_thousand_kwh" validate:"gte=0"`
}

// BulkUpsertPlanRequest wraps a non-empty slice of UpsertPlanRequest.
type BulkUpsertPlanRequest struct {
	Plans []UpsertPlanRequest `json:"plans" validate:"required,min=1,dive"`
}
