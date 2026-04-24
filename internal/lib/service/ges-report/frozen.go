package gesreportservice

import (
	model "srmt-admin/internal/lib/model/ges-report"
)

// applyFrozenSlice returns a new slice where each row has its frozen-default
// fallbacks applied. Rows whose org_id has no frozen entry are returned
// unchanged.
//
// Operates on a copy so the original repo-returned slice is not mutated.
func applyFrozenSlice(rows []model.RawDailyRow, frozen map[int64]map[string]float64) []model.RawDailyRow {
	if len(frozen) == 0 || len(rows) == 0 {
		return rows
	}
	out := make([]model.RawDailyRow, len(rows))
	for i, row := range rows {
		fields, ok := frozen[row.OrganizationID]
		if !ok || len(fields) == 0 {
			out[i] = row
			continue
		}
		out[i] = applyFrozenFallbacks(row, fields)
	}
	return out
}

// applyFrozenFallbacks overlays per-org frozen defaults onto a RawDailyRow.
//
// For nullable pointer fields (water_level_m, water_volume_mln_m3, water_head_m,
// reservoir_income_m3s, total_outflow_m3s, ges_flow_m3s): if row.X is nil and
// the field is frozen for this org, sets row.X to a pointer to the frozen value.
//
// For NOT NULL fields with COALESCE(0) semantics
// (daily_production_mln_kwh, working_aggregates, repair_aggregates,
// modernization_aggregates): the frozen value is applied ONLY when
// !row.HasRowForDate (i.e. no daily_data row exists for this org+date).
// When the row exists, an explicit 0 from the user is respected — frozen
// is NOT applied. See plan §2.7.
func applyFrozenFallbacks(row model.RawDailyRow, frozen map[string]float64) model.RawDailyRow {
	for field, value := range frozen {
		switch field {
		// --- Nullable pointer fields: apply only when row.X is nil. ---
		case model.FrozenFieldWaterLevelM:
			if row.WaterLevelM == nil {
				v := value
				row.WaterLevelM = &v
			}
		case model.FrozenFieldWaterVolumeMlnM3:
			if row.WaterVolumeMlnM3 == nil {
				v := value
				row.WaterVolumeMlnM3 = &v
			}
		case model.FrozenFieldWaterHeadM:
			if row.WaterHeadM == nil {
				v := value
				row.WaterHeadM = &v
			}
		case model.FrozenFieldReservoirIncomeM3s:
			if row.ReservoirIncomeM3s == nil {
				v := value
				row.ReservoirIncomeM3s = &v
			}
		case model.FrozenFieldTotalOutflowM3s:
			if row.TotalOutflowM3s == nil {
				v := value
				row.TotalOutflowM3s = &v
			}
		case model.FrozenFieldGESFlowM3s:
			if row.GESFlowM3s == nil {
				v := value
				row.GESFlowM3s = &v
			}

		// --- NOT NULL / COALESCE(0) fields: apply only when no row exists. ---
		// When HasRowForDate=true an explicit 0 is a user choice and must win
		// (plan §2.7); the frozen value is intentionally ignored.
		case model.FrozenFieldDailyProduction:
			if !row.HasRowForDate {
				row.DailyProductionMlnKWh = value
			}
		case model.FrozenFieldWorkingAggregates:
			if !row.HasRowForDate {
				row.WorkingAggregates = int(value)
			}
		case model.FrozenFieldRepairAggregates:
			if !row.HasRowForDate {
				row.RepairAggregates = int(value)
			}
		case model.FrozenFieldModernizationAggregates:
			if !row.HasRowForDate {
				row.ModernizationAggregates = int(value)
			}
		}
	}
	return row
}
