package gesreport

import (
	optional "srmt-admin/internal/lib/optional"
)

// --- Config ---

type Config struct {
	ID                   int64   `json:"id"`
	OrganizationID       int64   `json:"organization_id"`
	OrganizationName     string  `json:"organization_name,omitempty"`
	CascadeID            *int64  `json:"cascade_id,omitempty"`
	CascadeName          *string `json:"cascade_name,omitempty"`
	InstalledCapacityMWt float64 `json:"installed_capacity_mwt"`
	TotalAggregates      int     `json:"total_aggregates"`
	HasReservoir         bool    `json:"has_reservoir"`
	SortOrder            int     `json:"sort_order"`
}

type UpsertConfigRequest struct {
	OrganizationID       int64   `json:"organization_id" validate:"required"`
	InstalledCapacityMWt float64 `json:"installed_capacity_mwt" validate:"gte=0"`
	TotalAggregates      int     `json:"total_aggregates" validate:"gte=0"`
	HasReservoir         bool    `json:"has_reservoir"`
	SortOrder            int     `json:"sort_order" validate:"gte=0"`
}

// --- Cascade Config ---

type CascadeConfig struct {
	ID               int64    `json:"id"`
	OrganizationID   int64    `json:"organization_id"`
	OrganizationName string   `json:"organization_name,omitempty"`
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`
	SortOrder        int      `json:"sort_order"`
}

type UpsertCascadeConfigRequest struct {
	OrganizationID int64    `json:"organization_id" validate:"required"`
	Latitude       *float64 `json:"latitude,omitempty" validate:"omitempty,gte=-90,lte=90"`
	Longitude      *float64 `json:"longitude,omitempty" validate:"omitempty,gte=-180,lte=180"`
	SortOrder      int      `json:"sort_order" validate:"gte=0"`
}

// --- Daily Data ---

type DailyData struct {
	ID                      int64    `json:"id"`
	OrganizationID          int64    `json:"organization_id"`
	Date                    string   `json:"date"`
	DailyProductionMlnKWh   float64  `json:"daily_production_mln_kwh"`
	WorkingAggregates       int      `json:"working_aggregates"`
	RepairAggregates        int      `json:"repair_aggregates"`
	ModernizationAggregates int      `json:"modernization_aggregates"`
	WaterLevelM             *float64 `json:"water_level_m"`
	WaterVolumeMlnM3        *float64 `json:"water_volume_mln_m3"`
	WaterHeadM              *float64 `json:"water_head_m"`
	ReservoirIncomeM3s      *float64 `json:"reservoir_income_m3s"`
	TotalOutflowM3s         *float64 `json:"total_outflow_m3s"`
	GESFlowM3s              *float64 `json:"ges_flow_m3s"`
}

type UpsertDailyDataRequest struct {
	OrganizationID          int64                      `json:"organization_id" validate:"required"`
	Date                    string                     `json:"date" validate:"required"`
	DailyProductionMlnKWh   optional.Optional[float64] `json:"daily_production_mln_kwh"`
	WorkingAggregates       optional.Optional[int]     `json:"working_aggregates"`
	RepairAggregates        optional.Optional[int]     `json:"repair_aggregates"`
	ModernizationAggregates optional.Optional[int]     `json:"modernization_aggregates"`
	WaterLevelM             optional.Optional[float64] `json:"water_level_m"`
	WaterVolumeMlnM3        optional.Optional[float64] `json:"water_volume_mln_m3"`
	WaterHeadM              optional.Optional[float64] `json:"water_head_m"`
	ReservoirIncomeM3s      optional.Optional[float64] `json:"reservoir_income_m3s"`
	TotalOutflowM3s         optional.Optional[float64] `json:"total_outflow_m3s"`
	GESFlowM3s              optional.Optional[float64] `json:"ges_flow_m3s"`
}

// UpsertCascadeDailyWeatherRequest is the body item for manual weather corrections
// on a cascade organization. The handler writes to cascade_daily_data using
// three-state Optional semantics: absent preserves the column, null writes NULL,
// a value writes that value.
type UpsertCascadeDailyWeatherRequest struct {
	OrganizationID   int64                      `json:"organization_id" validate:"required"`
	Date             string                     `json:"date" validate:"required"`
	Temperature      optional.Optional[float64] `json:"temperature"`
	WeatherCondition optional.Optional[string]  `json:"weather_condition"`
}

// --- Production Plan ---

type ProductionPlan struct {
	ID             int64   `json:"id"`
	OrganizationID int64   `json:"organization_id"`
	Year           int     `json:"year"`
	Month          int     `json:"month"`
	PlanMlnKWh     float64 `json:"plan_mln_kwh"`
}

type UpsertPlanRequest struct {
	OrganizationID int64   `json:"organization_id" validate:"required"`
	Year           int     `json:"year" validate:"required,gte=2020,lte=2100"`
	Month          int     `json:"month" validate:"required,gte=1,lte=12"`
	PlanMlnKWh     float64 `json:"plan_mln_kwh" validate:"gte=0"`
}

type BulkUpsertPlanRequest struct {
	Plans []UpsertPlanRequest `json:"plans" validate:"required,min=1,dive"`
}

// --- Report Response ---

type DailyReport struct {
	Date       string          `json:"date"`
	Cascades   []CascadeReport `json:"cascades"`
	GrandTotal *SummaryBlock   `json:"grand_total"`
}

type CascadeReport struct {
	CascadeID   int64           `json:"cascade_id"`
	CascadeName string          `json:"cascade_name"`
	Weather     *CascadeWeather `json:"weather"`
	Summary     *SummaryBlock   `json:"summary"`
	Stations    []StationReport `json:"stations"`
}

// CascadeWeather holds per-cascade weather for the report response.
// Populated by the service from cascade_daily_data (not from station rows).
type CascadeWeather struct {
	Temperature         *float64 `json:"temperature"`
	Condition           *string  `json:"weather_condition"`
	PrevYearTemperature *float64 `json:"prev_year_temperature"`
	PrevYearCondition   *string  `json:"prev_year_condition"`
}

// CascadeWeatherKey is the map key for batch weather lookups by (cascade org id, date).
// Must remain comparable — do not add pointer fields.
type CascadeWeatherKey struct {
	OrgID int64
	Date  string
}

type StationReport struct {
	OrganizationID int64              `json:"organization_id"`
	Name           string             `json:"name"`
	Config         StationConfig      `json:"config"`
	Current        CurrentData        `json:"current"`
	PreviousDay    *PreviousDayData   `json:"previous_day"`
	Diffs          DiffData           `json:"diffs"`
	Aggregations   Aggregations       `json:"aggregations"`
	Plan           PlanData           `json:"plan"`
	PreviousYear   *PrevYearData      `json:"previous_year"`
	YoY            YoYData            `json:"yoy"`
	IdleDischarge  *IdleDischargeData `json:"idle_discharge"`
}

type StationConfig struct {
	InstalledCapacityMWt float64 `json:"installed_capacity_mwt"`
	TotalAggregates      int     `json:"total_aggregates"`
	HasReservoir         bool    `json:"has_reservoir"`
}

type CurrentData struct {
	DailyProductionMlnKWh   float64  `json:"daily_production_mln_kwh"`
	PowerMWt                float64  `json:"power_mwt"`
	WorkingAggregates       int      `json:"working_aggregates"`
	RepairAggregates        int      `json:"repair_aggregates"`
	ModernizationAggregates int      `json:"modernization_aggregates"`
	ReserveAggregates       int      `json:"reserve_aggregates"`
	WaterLevelM             *float64 `json:"water_level_m"`
	WaterVolumeMlnM3        *float64 `json:"water_volume_mln_m3"`
	WaterHeadM              *float64 `json:"water_head_m"`
	ReservoirIncomeM3s      *float64 `json:"reservoir_income_m3s"`
	TotalOutflowM3s         *float64 `json:"total_outflow_m3s"`
	GESFlowM3s              *float64 `json:"ges_flow_m3s"`
	IdleDischargeM3s        *float64 `json:"idle_discharge_m3s"`
}

// PreviousDayData is a snapshot of a station's state for the previous
// operational day. It MUST stay structurally identical to CurrentData (same
// fields, types, json tags, in the same order) so the service can convert
// via PreviousDayData(currentData) cast — see service.computeDaySnapshot.
// The distinct type keeps semantic meaning visible at field sites and
// prevents accidental mixing.
type PreviousDayData struct {
	DailyProductionMlnKWh   float64  `json:"daily_production_mln_kwh"`
	PowerMWt                float64  `json:"power_mwt"`
	WorkingAggregates       int      `json:"working_aggregates"`
	RepairAggregates        int      `json:"repair_aggregates"`
	ModernizationAggregates int      `json:"modernization_aggregates"`
	ReserveAggregates       int      `json:"reserve_aggregates"`
	WaterLevelM             *float64 `json:"water_level_m"`
	WaterVolumeMlnM3        *float64 `json:"water_volume_mln_m3"`
	WaterHeadM              *float64 `json:"water_head_m"`
	ReservoirIncomeM3s      *float64 `json:"reservoir_income_m3s"`
	TotalOutflowM3s         *float64 `json:"total_outflow_m3s"`
	GESFlowM3s              *float64 `json:"ges_flow_m3s"`
	IdleDischargeM3s        *float64 `json:"idle_discharge_m3s"`
}

type DiffData struct {
	LevelChangeCm     *float64 `json:"level_change_cm"`
	VolumeChangeMlnM3 *float64 `json:"volume_change_mln_m3"`
	IncomeChangeM3s   *float64 `json:"income_change_m3s"`
	GESFlowChangeM3s  *float64 `json:"ges_flow_change_m3s"`
	PowerChangeMWt    *float64 `json:"power_change_mwt"`
	ProductionChange  *float64 `json:"production_change_mln_kwh"`
}

type Aggregations struct {
	MTDProductionMlnKWh float64 `json:"mtd_production_mln_kwh"`
	YTDProductionMlnKWh float64 `json:"ytd_production_mln_kwh"`
}

type PlanData struct {
	MonthlyPlanMlnKWh   float64  `json:"monthly_plan_mln_kwh"`
	QuarterlyPlanMlnKWh float64  `json:"quarterly_plan_mln_kwh"`
	FulfillmentPct      *float64 `json:"fulfillment_pct"`
	DifferenceMlnKWh    float64  `json:"difference_mln_kwh"`
}

type PrevYearData struct {
	WaterLevelM        *float64 `json:"water_level_m"`
	WaterVolumeMlnM3   *float64 `json:"water_volume_mln_m3"`
	WaterHeadM         *float64 `json:"water_head_m"`
	ReservoirIncomeM3s *float64 `json:"reservoir_income_m3s"`
	GESFlowM3s         *float64 `json:"ges_flow_m3s"`
	PowerMWt           *float64 `json:"power_mwt"`
	DailyProduction    *float64 `json:"daily_production_mln_kwh"`
	MTDProduction      float64  `json:"mtd_production_mln_kwh"`
	YTDProduction      float64  `json:"ytd_production_mln_kwh"`
}

type YoYData struct {
	GrowthRate       *float64 `json:"growth_rate"`
	DifferenceMlnKWh float64  `json:"difference_mln_kwh"`
}

type IdleDischargeData struct {
	FlowRateM3s float64 `json:"flow_rate_m3s"`
	VolumeMlnM3 float64 `json:"volume_mln_m3"`
	Reason      *string `json:"reason"`
	IsOngoing   bool    `json:"is_ongoing"`
}

// SummaryBlock is used for cascade totals and grand total.
type SummaryBlock struct {
	InstalledCapacityMWt    float64  `json:"installed_capacity_mwt"`
	TotalAggregates         int      `json:"total_aggregates"`
	WorkingAggregates       int      `json:"working_aggregates"`
	RepairAggregates        int      `json:"repair_aggregates"`
	ModernizationAggregates int      `json:"modernization_aggregates"`
	ReserveAggregates       int      `json:"reserve_aggregates"`
	PowerMWt                float64  `json:"power_mwt"`
	DailyProductionMlnKWh   float64  `json:"daily_production_mln_kwh"`
	ProductionChange        float64  `json:"production_change_mln_kwh"`
	MTDProductionMlnKWh     float64  `json:"mtd_production_mln_kwh"`
	YTDProductionMlnKWh     float64  `json:"ytd_production_mln_kwh"`
	MonthlyPlanMlnKWh       float64  `json:"monthly_plan_mln_kwh"`
	QuarterlyPlanMlnKWh     float64  `json:"quarterly_plan_mln_kwh"`
	FulfillmentPct          *float64 `json:"fulfillment_pct"`
	DifferenceMlnKWh        float64  `json:"difference_mln_kwh"`
	PrevYearYTD             float64  `json:"prev_year_ytd_mln_kwh"`
	YoYGrowthRate           *float64 `json:"yoy_growth_rate"`
	YoYDifference           float64  `json:"yoy_difference_mln_kwh"`
	IdleDischargeM3s        float64  `json:"idle_discharge_total_m3s"`
}

// --- Internal query structs (used by repo) ---

type RawDailyRow struct {
	OrganizationID          int64
	OrganizationName        string
	CascadeID               *int64
	CascadeName             *string
	Date                    string
	DailyProductionMlnKWh   float64
	WorkingAggregates       int
	RepairAggregates        int
	ModernizationAggregates int
	WaterLevelM             *float64
	WaterVolumeMlnM3        *float64
	WaterHeadM              *float64
	ReservoirIncomeM3s      *float64
	TotalOutflowM3s         *float64
	GESFlowM3s              *float64
	InstalledCapacityMWt    float64
	TotalAggregates         int
	HasReservoir            bool
	SortOrder               int
}

type ProductionAggregation struct {
	OrganizationID int64
	MTD            float64
	YTD            float64
	PrevYearMTD    float64
	PrevYearYTD    float64
}

type PlanRow struct {
	OrganizationID int64   `json:"organization_id"`
	Year           int     `json:"year"`
	Month          int     `json:"month"`
	PlanMlnKWh     float64 `json:"plan_mln_kwh"`
}

type IdleDischargeRow struct {
	OrganizationID int64
	FlowRateM3s    float64
	VolumeMlnM3    float64
	Reason         *string
	IsOngoing      bool
}

// AggregateCounts is a tuple of the three persisted aggregate counters used by
// handler-level validation to fold the request onto the current DB row before
// checking sum ≤ ges_config.total_aggregates.
type AggregateCounts struct {
	Working       int
	Repair        int
	Modernization int
}

// --- Helper functions ---

// QuarterMonths returns the 3 months of the quarter containing the given month.
func QuarterMonths(month int) []int {
	switch {
	case month <= 3:
		return []int{1, 2, 3}
	case month <= 6:
		return []int{4, 5, 6}
	case month <= 9:
		return []int{7, 8, 9}
	default:
		return []int{10, 11, 12}
	}
}

// NullableDiff returns pointer to (a - b) if both non-nil, else nil.
func NullableDiff(a, b *float64) *float64 {
	if a == nil || b == nil {
		return nil
	}
	v := *a - *b
	return &v
}

// SafeDiv returns a/b if b != 0, else nil.
func SafeDiv(a, b float64) *float64 {
	if b == 0 {
		return nil
	}
	v := a / b
	return &v
}
