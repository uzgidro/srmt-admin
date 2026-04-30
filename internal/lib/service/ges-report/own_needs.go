package gesreportservice

import (
	"context"
	"fmt"

	model "srmt-admin/internal/lib/model/ges-report"
)

// BuildOwnNeedsReport assembles a compact report focused on own/economic
// consumption (СН/ХН) for the dedicated Excel export. It reuses BuildDailyReport
// — no extra SQL — and re-shapes the resulting DailyReport into the smaller
// OwnNeedsReport DTO. Stations within a cascade preserve the order returned by
// the upstream service (sort_order, then name).
func (s *Service) BuildOwnNeedsReport(ctx context.Context, date string) (*model.OwnNeedsReport, error) {
	full, err := s.BuildDailyReport(ctx, date, nil)
	if err != nil {
		return nil, fmt.Errorf("BuildOwnNeedsReport: %w", err)
	}

	report := &model.OwnNeedsReport{
		Date:     full.Date,
		Cascades: make([]model.OwnNeedsCascade, 0, len(full.Cascades)),
	}

	for _, c := range full.Cascades {
		cas := model.OwnNeedsCascade{
			CascadeID:   c.CascadeID,
			CascadeName: c.CascadeName,
			Stations:    make([]model.OwnNeedsStation, 0, len(c.Stations)),
		}
		for _, st := range c.Stations {
			ownDelta := computeOwnDelta(st)
			cas.Stations = append(cas.Stations, model.OwnNeedsStation{
				OrganizationID:        st.OrganizationID,
				Name:                  st.Name,
				InstalledCapacityMWt:  st.Config.InstalledCapacityMWt,
				MonthlyPlanMlnKWh:     st.Plan.MonthlyPlanMlnKWh,
				CumulativePlanMlnKWh:  st.Plan.QuarterlyPlanMlnKWh,
				DailyProductionMlnKWh: st.Current.DailyProductionMlnKWh,
				DailyProductionDelta:  st.Diffs.ProductionChange,
				MTDProductionMlnKWh:   st.Aggregations.MTDProductionMlnKWh,
				YTDProductionMlnKWh:   st.Aggregations.YTDProductionMlnKWh,
				OwnConsumptionKWh:     st.Current.OwnConsumptionKWh,
				OwnConsumptionDelta:   ownDelta,
				MTDOwnConsumptionKWh:  st.Aggregations.MTDOwnConsumptionKWh,
				YTDOwnConsumptionKWh:  st.Aggregations.YTDOwnConsumptionKWh,
			})
		}
		cas.Totals = sumOwnNeedsStations(cas.Stations)
		report.Cascades = append(report.Cascades, cas)
	}

	report.GrandTotal = sumOwnNeedsCascades(report.Cascades)
	return report, nil
}

// computeOwnDelta returns today.OwnConsumptionKWh − yesterday.OwnConsumptionKWh
// when both are present, else nil. The yesterday value comes from the
// PreviousDay snapshot the upstream service already populates.
func computeOwnDelta(st model.StationReport) *float64 {
	if st.PreviousDay == nil {
		return nil
	}
	return model.NullableDiff(st.Current.OwnConsumptionKWh, st.PreviousDay.OwnConsumptionKWh)
}

// sumOwnNeedsStations folds station-level fields into a totals block.
// Nil pointer fields contribute 0 (matches the sample report behaviour where
// missing values display as blank/0).
func sumOwnNeedsStations(stations []model.OwnNeedsStation) model.OwnNeedsTotals {
	var t model.OwnNeedsTotals
	for _, st := range stations {
		t.InstalledCapacityMWt += st.InstalledCapacityMWt
		t.MonthlyPlanMlnKWh += st.MonthlyPlanMlnKWh
		t.CumulativePlanMlnKWh += st.CumulativePlanMlnKWh
		t.DailyProductionMlnKWh += st.DailyProductionMlnKWh
		if st.DailyProductionDelta != nil {
			t.DailyProductionDelta += *st.DailyProductionDelta
		}
		t.MTDProductionMlnKWh += st.MTDProductionMlnKWh
		t.YTDProductionMlnKWh += st.YTDProductionMlnKWh
		if st.OwnConsumptionKWh != nil {
			t.OwnConsumptionKWh += *st.OwnConsumptionKWh
		}
		if st.OwnConsumptionDelta != nil {
			t.OwnConsumptionDelta += *st.OwnConsumptionDelta
		}
		t.MTDOwnConsumptionKWh += st.MTDOwnConsumptionKWh
		t.YTDOwnConsumptionKWh += st.YTDOwnConsumptionKWh
	}
	return t
}

// sumOwnNeedsCascades folds the cascade totals into a grand total.
func sumOwnNeedsCascades(cascades []model.OwnNeedsCascade) model.OwnNeedsTotals {
	var t model.OwnNeedsTotals
	for _, c := range cascades {
		t.InstalledCapacityMWt += c.Totals.InstalledCapacityMWt
		t.MonthlyPlanMlnKWh += c.Totals.MonthlyPlanMlnKWh
		t.CumulativePlanMlnKWh += c.Totals.CumulativePlanMlnKWh
		t.DailyProductionMlnKWh += c.Totals.DailyProductionMlnKWh
		t.DailyProductionDelta += c.Totals.DailyProductionDelta
		t.MTDProductionMlnKWh += c.Totals.MTDProductionMlnKWh
		t.YTDProductionMlnKWh += c.Totals.YTDProductionMlnKWh
		t.OwnConsumptionKWh += c.Totals.OwnConsumptionKWh
		t.OwnConsumptionDelta += c.Totals.OwnConsumptionDelta
		t.MTDOwnConsumptionKWh += c.Totals.MTDOwnConsumptionKWh
		t.YTDOwnConsumptionKWh += c.Totals.YTDOwnConsumptionKWh
	}
	return t
}
