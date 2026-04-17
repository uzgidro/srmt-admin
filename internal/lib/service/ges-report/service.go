package gesreportservice

import (
	"context"
	"fmt"
	"time"

	model "srmt-admin/internal/lib/model/ges-report"
)

// Repository defines data-access methods required by the service.
type Repository interface {
	GetGESDailyDataBatch(ctx context.Context, date string) ([]model.RawDailyRow, error)
	GetGESProductionAggregations(ctx context.Context, date string) ([]model.ProductionAggregation, error)
	GetGESPlansForReport(ctx context.Context, year int, months []int) ([]model.PlanRow, error)
	GetIdleDischargesForDate(ctx context.Context, start, end time.Time) ([]model.IdleDischargeRow, error)
	GetCascadeDailyWeatherBatch(ctx context.Context, orgIDs []int64, dates []string) (map[model.CascadeWeatherKey]*model.CascadeWeather, error)
}

// Service assembles the GES daily report.
type Service struct {
	repo Repository
	loc  *time.Location
}

// NewService creates a new Service.
func NewService(repo Repository, loc *time.Location) *Service {
	return &Service{repo: repo, loc: loc}
}

// BuildDailyReport assembles the full GES daily report for the given date string (YYYY-MM-DD).
// If cascadeOrgID is non-nil, the report is filtered to only include the cascade matching that
// org ID, and GrandTotal is recomputed as the sum of just that cascade's stations.
func (s *Service) BuildDailyReport(ctx context.Context, date string, cascadeOrgID *int64) (*model.DailyReport, error) {
	// 1. Parse date and compute related dates.
	t, err := time.ParseInLocation("2006-01-02", date, s.loc)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", date, err)
	}

	yesterday := t.AddDate(0, 0, -1).Format("2006-01-02")
	prevYear := t.AddDate(-1, 0, 0).Format("2006-01-02")

	year := t.Year()
	month := int(t.Month())
	quarterMonths := model.QuarterMonths(month)

	// 2. Operational day boundaries: 05:00 today → 05:00 tomorrow.
	dayStart := time.Date(t.Year(), t.Month(), t.Day(), 5, 0, 0, 0, s.loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	// 3. Fetch all data in parallel (sequentially for simplicity; repos are DB-bound).
	todayData, err := s.repo.GetGESDailyDataBatch(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("GetGESDailyDataBatch(today): %w", err)
	}
	yesterdayData, err := s.repo.GetGESDailyDataBatch(ctx, yesterday)
	if err != nil {
		return nil, fmt.Errorf("GetGESDailyDataBatch(yesterday): %w", err)
	}
	prevYearData, err := s.repo.GetGESDailyDataBatch(ctx, prevYear)
	if err != nil {
		return nil, fmt.Errorf("GetGESDailyDataBatch(prevYear): %w", err)
	}
	aggregations, err := s.repo.GetGESProductionAggregations(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("GetGESProductionAggregations: %w", err)
	}
	plans, err := s.repo.GetGESPlansForReport(ctx, year, quarterMonths)
	if err != nil {
		return nil, fmt.Errorf("GetGESPlansForReport: %w", err)
	}
	discharges, err := s.repo.GetIdleDischargesForDate(ctx, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("GetIdleDischargesForDate: %w", err)
	}

	// Collect unique cascade org IDs from todayData for batch weather lookup.
	cascadeOrgIDSet := make(map[int64]struct{})
	cascadeOrgIDs := make([]int64, 0)
	for _, row := range todayData {
		if row.CascadeID != nil {
			if _, seen := cascadeOrgIDSet[*row.CascadeID]; !seen {
				cascadeOrgIDSet[*row.CascadeID] = struct{}{}
				cascadeOrgIDs = append(cascadeOrgIDs, *row.CascadeID)
			}
		}
	}

	weatherToday, err := s.repo.GetCascadeDailyWeatherBatch(ctx, cascadeOrgIDs, []string{date})
	if err != nil {
		return nil, fmt.Errorf("GetCascadeDailyWeatherBatch(today): %w", err)
	}
	weatherPrevYear, err := s.repo.GetCascadeDailyWeatherBatch(ctx, cascadeOrgIDs, []string{prevYear})
	if err != nil {
		return nil, fmt.Errorf("GetCascadeDailyWeatherBatch(prevYear): %w", err)
	}

	// 4. Build lookup maps.
	yesterdayMap := buildRawMap(yesterdayData)
	prevYearMap := buildRawMap(prevYearData)
	aggMap := buildAggMap(aggregations)
	planMap := buildPlanMap(plans, month)
	dischargeMap := buildDischargeMap(discharges)

	// 5. Compute per-station reports, grouped by cascade.
	type cascadeKey struct {
		id   int64
		name string
	}
	cascadeOrder := []cascadeKey{}
	cascadeStations := map[cascadeKey][]model.StationReport{}

	for _, row := range todayData {
		station := s.computeStation(row, yesterdayMap, prevYearMap, aggMap, planMap, dischargeMap)

		var cid int64
		var cname string
		if row.CascadeID != nil {
			cid = *row.CascadeID
		}
		if row.CascadeName != nil {
			cname = *row.CascadeName
		}

		key := cascadeKey{id: cid, name: cname}
		if _, exists := cascadeStations[key]; !exists {
			cascadeOrder = append(cascadeOrder, key)
		}
		cascadeStations[key] = append(cascadeStations[key], station)
	}

	// 6. Build cascade reports.
	cascades := make([]model.CascadeReport, 0, len(cascadeOrder))
	for _, key := range cascadeOrder {
		stations := cascadeStations[key]
		summary := computeSummary(stations)
		cascades = append(cascades, model.CascadeReport{
			CascadeID:   key.id,
			CascadeName: key.name,
			Weather:     buildCascadeWeather(key.id, date, prevYear, weatherToday, weatherPrevYear),
			Summary:     summary,
			Stations:    stations,
		})
	}

	// 7. Optional cascade filter: restrict to a single cascade by org ID.
	if cascadeOrgID != nil {
		filtered := make([]model.CascadeReport, 0, 1)
		for _, c := range cascades {
			if c.CascadeID == *cascadeOrgID {
				filtered = append(filtered, c)
				break
			}
		}
		cascades = filtered
	}

	// 8. Grand total (computed over the possibly-filtered cascade slice).
	grandTotal := computeGrandTotal(cascades)

	return &model.DailyReport{
		Date:       date,
		Cascades:   cascades,
		GrandTotal: grandTotal,
	}, nil
}

// computeStation builds a StationReport from a single today row and lookup maps.
func (s *Service) computeStation(
	row model.RawDailyRow,
	yesterdayMap map[int64]model.RawDailyRow,
	prevYearMap map[int64]model.RawDailyRow,
	aggMap map[int64]model.ProductionAggregation,
	planMap map[int64]planEntry,
	dischargeMap map[int64]model.IdleDischargeData,
) model.StationReport {
	// Power from daily production.
	power := row.DailyProductionMlnKWh * 1000.0 / 24.0

	// Idle discharge = totalOutflow - gesFlow (both nullable).
	var idleM3s *float64
	if row.TotalOutflowM3s != nil && row.GESFlowM3s != nil {
		v := *row.TotalOutflowM3s - *row.GESFlowM3s
		idleM3s = &v
	}

	current := model.CurrentData{
		DailyProductionMlnKWh: row.DailyProductionMlnKWh,
		PowerMWt:              power,
		WorkingAggregates:     row.WorkingAggregates,
		WaterLevelM:           row.WaterLevelM,
		WaterVolumeMlnM3:      row.WaterVolumeMlnM3,
		WaterHeadM:            row.WaterHeadM,
		ReservoirIncomeM3s:    row.ReservoirIncomeM3s,
		TotalOutflowM3s:       row.TotalOutflowM3s,
		GESFlowM3s:            row.GESFlowM3s,
		IdleDischargeM3s:      idleM3s,
	}

	// Diffs vs yesterday.
	var diffs model.DiffData
	if yest, ok := yesterdayMap[row.OrganizationID]; ok {
		// Level change in cm (multiply by 100).
		if row.WaterLevelM != nil && yest.WaterLevelM != nil {
			v := (*row.WaterLevelM - *yest.WaterLevelM) * 100.0
			diffs.LevelChangeCm = &v
		}
		diffs.VolumeChangeMlnM3 = model.NullableDiff(row.WaterVolumeMlnM3, yest.WaterVolumeMlnM3)
		diffs.IncomeChangeM3s = model.NullableDiff(row.ReservoirIncomeM3s, yest.ReservoirIncomeM3s)
		diffs.GESFlowChangeM3s = model.NullableDiff(row.GESFlowM3s, yest.GESFlowM3s)

		yestPower := yest.DailyProductionMlnKWh * 1000.0 / 24.0
		powerChange := power - yestPower
		diffs.PowerChangeMWt = &powerChange

		prodChange := row.DailyProductionMlnKWh - yest.DailyProductionMlnKWh
		diffs.ProductionChange = &prodChange
	}

	// Aggregations.
	agg := aggMap[row.OrganizationID]
	aggregations := model.Aggregations{
		MTDProductionMlnKWh: agg.MTD,
		YTDProductionMlnKWh: agg.YTD,
	}

	// Plan.
	pe := planMap[row.OrganizationID]
	ytd := agg.YTD
	fulfillment := model.SafeDiv(ytd, pe.quarterly)
	planData := model.PlanData{
		MonthlyPlanMlnKWh:   pe.monthly,
		QuarterlyPlanMlnKWh: pe.quarterly,
		FulfillmentPct:      fulfillment,
		DifferenceMlnKWh:    ytd - pe.quarterly,
	}

	// Previous year.
	var prevYear *model.PrevYearData
	if py, ok := prevYearMap[row.OrganizationID]; ok {
		pyPower := py.DailyProductionMlnKWh * 1000.0 / 24.0
		prevYear = &model.PrevYearData{
			WaterLevelM:        py.WaterLevelM,
			WaterVolumeMlnM3:   py.WaterVolumeMlnM3,
			WaterHeadM:         py.WaterHeadM,
			ReservoirIncomeM3s: py.ReservoirIncomeM3s,
			GESFlowM3s:         py.GESFlowM3s,
			PowerMWt:           &pyPower,
			DailyProduction:    &py.DailyProductionMlnKWh,
			MTDProduction:      agg.PrevYearMTD,
			YTDProduction:      agg.PrevYearYTD,
		}
	}

	// YoY.
	prevYearYTD := agg.PrevYearYTD
	var yoy model.YoYData
	if prevYearYTD != 0 {
		rate := model.SafeDiv(ytd, prevYearYTD)
		if rate != nil {
			adjusted := *rate - 1.0
			yoy.GrowthRate = &adjusted
		}
	}
	yoy.DifferenceMlnKWh = ytd - prevYearYTD

	// Idle discharge from discharge map.
	var idleDischarge *model.IdleDischargeData
	if d, ok := dischargeMap[row.OrganizationID]; ok {
		dc := d
		idleDischarge = &dc
	}

	return model.StationReport{
		OrganizationID: row.OrganizationID,
		Name:           row.OrganizationName,
		Config: model.StationConfig{
			InstalledCapacityMWt: row.InstalledCapacityMWt,
			TotalAggregates:      row.TotalAggregates,
			HasReservoir:         row.HasReservoir,
		},
		Current:       current,
		Diffs:         diffs,
		Aggregations:  aggregations,
		Plan:          planData,
		PreviousYear:  prevYear,
		YoY:           yoy,
		IdleDischarge: idleDischarge,
	}
}

// computeSummary sums the relevant fields across stations in a cascade.
func computeSummary(stations []model.StationReport) *model.SummaryBlock {
	sb := &model.SummaryBlock{}
	for _, st := range stations {
		sb.InstalledCapacityMWt += st.Config.InstalledCapacityMWt
		sb.TotalAggregates += st.Config.TotalAggregates
		sb.WorkingAggregates += st.Current.WorkingAggregates
		sb.PowerMWt += st.Current.PowerMWt
		sb.DailyProductionMlnKWh += st.Current.DailyProductionMlnKWh
		if st.Diffs.ProductionChange != nil {
			sb.ProductionChange += *st.Diffs.ProductionChange
		}
		sb.MTDProductionMlnKWh += st.Aggregations.MTDProductionMlnKWh
		sb.YTDProductionMlnKWh += st.Aggregations.YTDProductionMlnKWh
		sb.MonthlyPlanMlnKWh += st.Plan.MonthlyPlanMlnKWh
		sb.QuarterlyPlanMlnKWh += st.Plan.QuarterlyPlanMlnKWh
		sb.PrevYearYTD += func() float64 {
			if st.PreviousYear != nil {
				return st.PreviousYear.YTDProduction
			}
			return 0
		}()
		if st.IdleDischarge != nil {
			sb.IdleDischargeM3s += st.IdleDischarge.FlowRateM3s
		}
	}

	// Derived fields.
	sb.FulfillmentPct = model.SafeDiv(sb.YTDProductionMlnKWh, sb.QuarterlyPlanMlnKWh)
	sb.DifferenceMlnKWh = sb.YTDProductionMlnKWh - sb.QuarterlyPlanMlnKWh
	if sb.PrevYearYTD != 0 {
		rate := model.SafeDiv(sb.YTDProductionMlnKWh, sb.PrevYearYTD)
		if rate != nil {
			adjusted := *rate - 1.0
			sb.YoYGrowthRate = &adjusted
		}
	}
	sb.YoYDifference = sb.YTDProductionMlnKWh - sb.PrevYearYTD

	return sb
}

// computeGrandTotal sums cascade summaries and computes derived fields.
func computeGrandTotal(cascades []model.CascadeReport) *model.SummaryBlock {
	gt := &model.SummaryBlock{}
	for _, c := range cascades {
		if c.Summary == nil {
			continue
		}
		s := c.Summary
		gt.InstalledCapacityMWt += s.InstalledCapacityMWt
		gt.TotalAggregates += s.TotalAggregates
		gt.WorkingAggregates += s.WorkingAggregates
		gt.PowerMWt += s.PowerMWt
		gt.DailyProductionMlnKWh += s.DailyProductionMlnKWh
		gt.ProductionChange += s.ProductionChange
		gt.MTDProductionMlnKWh += s.MTDProductionMlnKWh
		gt.YTDProductionMlnKWh += s.YTDProductionMlnKWh
		gt.MonthlyPlanMlnKWh += s.MonthlyPlanMlnKWh
		gt.QuarterlyPlanMlnKWh += s.QuarterlyPlanMlnKWh
		gt.PrevYearYTD += s.PrevYearYTD
		gt.IdleDischargeM3s += s.IdleDischargeM3s
	}

	// Derived fields.
	gt.FulfillmentPct = model.SafeDiv(gt.YTDProductionMlnKWh, gt.QuarterlyPlanMlnKWh)
	gt.DifferenceMlnKWh = gt.YTDProductionMlnKWh - gt.QuarterlyPlanMlnKWh
	if gt.PrevYearYTD != 0 {
		rate := model.SafeDiv(gt.YTDProductionMlnKWh, gt.PrevYearYTD)
		if rate != nil {
			adjusted := *rate - 1.0
			gt.YoYGrowthRate = &adjusted
		}
	}
	gt.YoYDifference = gt.YTDProductionMlnKWh - gt.PrevYearYTD

	return gt
}

// buildCascadeWeather composes a CascadeWeather for the given cascade from the
// today and previous-year batch lookup maps. Returns nil if no data is available.
func buildCascadeWeather(
	cascadeID int64,
	date, prevYear string,
	today, prevYr map[model.CascadeWeatherKey]*model.CascadeWeather,
) *model.CascadeWeather {
	wt := today[model.CascadeWeatherKey{OrgID: cascadeID, Date: date}]
	wp := prevYr[model.CascadeWeatherKey{OrgID: cascadeID, Date: prevYear}]
	if wt == nil && wp == nil {
		return nil
	}
	result := &model.CascadeWeather{}
	if wt != nil {
		result.Temperature = wt.Temperature
		result.Condition = wt.Condition
	}
	if wp != nil {
		result.PrevYearTemperature = wp.Temperature
		result.PrevYearCondition = wp.Condition
	}
	return result
}

// --- Lookup map builders ---

func buildRawMap(rows []model.RawDailyRow) map[int64]model.RawDailyRow {
	m := make(map[int64]model.RawDailyRow, len(rows))
	for _, r := range rows {
		m[r.OrganizationID] = r
	}
	return m
}

func buildAggMap(rows []model.ProductionAggregation) map[int64]model.ProductionAggregation {
	m := make(map[int64]model.ProductionAggregation, len(rows))
	for _, r := range rows {
		m[r.OrganizationID] = r
	}
	return m
}

// planEntry holds the monthly and quarterly plan values for one org.
type planEntry struct {
	monthly   float64
	quarterly float64
}

// buildPlanMap builds a map from orgID to planEntry.
// monthly is the plan for the current month; quarterly is the sum of all quarter-month plans.
func buildPlanMap(rows []model.PlanRow, currentMonth int) map[int64]planEntry {
	m := make(map[int64]planEntry)
	for _, r := range rows {
		pe := m[r.OrganizationID]
		pe.quarterly += r.PlanMlnKWh
		if r.Month == currentMonth {
			pe.monthly = r.PlanMlnKWh
		}
		m[r.OrganizationID] = pe
	}
	return m
}

// buildDischargeMap aggregates multiple discharge rows per org:
// sum volumes, derive average flow rate as totalVolume / 0.0864 (млн м³ → м³/с),
// keep first reason, IsOngoing=true if any is ongoing.
func buildDischargeMap(rows []model.IdleDischargeRow) map[int64]model.IdleDischargeData {
	const volumeToFlowRate = 0.0864 // 86400 с / 1 000 000 м³

	m := make(map[int64]model.IdleDischargeData)
	for _, r := range rows {
		existing, exists := m[r.OrganizationID]
		if !exists {
			m[r.OrganizationID] = model.IdleDischargeData{
				VolumeMlnM3: r.VolumeMlnM3,
				Reason:      r.Reason,
				IsOngoing:   r.IsOngoing,
			}
		} else {
			existing.VolumeMlnM3 += r.VolumeMlnM3
			if r.IsOngoing {
				existing.IsOngoing = true
			}
			m[r.OrganizationID] = existing
		}
	}

	for orgID, d := range m {
		if d.VolumeMlnM3 != 0 {
			d.FlowRateM3s = d.VolumeMlnM3 / volumeToFlowRate
		}
		m[orgID] = d
	}

	return m
}
