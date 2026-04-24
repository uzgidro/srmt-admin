package gesreport

// FreezableField name constants — single source of truth, mirrored by
// migration CHECK constraint and the validate:"oneof=..." struct tags.
const (
	FrozenFieldDailyProduction         = "daily_production_mln_kwh"
	FrozenFieldWorkingAggregates       = "working_aggregates"
	FrozenFieldRepairAggregates        = "repair_aggregates"
	FrozenFieldModernizationAggregates = "modernization_aggregates"
	FrozenFieldWaterLevelM             = "water_level_m"
	FrozenFieldWaterVolumeMlnM3        = "water_volume_mln_m3"
	FrozenFieldWaterHeadM              = "water_head_m"
	FrozenFieldReservoirIncomeM3s      = "reservoir_income_m3s"
	FrozenFieldTotalOutflowM3s         = "total_outflow_m3s"
	FrozenFieldGESFlowM3s              = "ges_flow_m3s"
)

// FreezableFields lists all field names that can be frozen — used by
// the service-layer applyFrozenFallbacks dispatch.
var FreezableFields = []string{
	FrozenFieldDailyProduction,
	FrozenFieldWorkingAggregates,
	FrozenFieldRepairAggregates,
	FrozenFieldModernizationAggregates,
	FrozenFieldWaterLevelM,
	FrozenFieldWaterVolumeMlnM3,
	FrozenFieldWaterHeadM,
	FrozenFieldReservoirIncomeM3s,
	FrozenFieldTotalOutflowM3s,
	FrozenFieldGESFlowM3s,
}

// IntegerFreezableFields are the subset that must be whole numbers
// (used by handler validation: frozen_value for these must equal Trunc(frozen_value)).
var IntegerFreezableFields = map[string]bool{
	FrozenFieldWorkingAggregates:       true,
	FrozenFieldRepairAggregates:        true,
	FrozenFieldModernizationAggregates: true,
}
