// Package sel transforms reservoir-flood-hourly records into a sel-generator
// Report. It collapses each reservoir's two RecordedAt rows (prev hour and
// current hour) into one ReservoirRow with paired Prev/Curr fields, ordered
// by reservoir_flood_config.sort_order.
package sel

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	floodmodel "srmt-admin/internal/lib/model/reservoir-flood"
	selgen "srmt-admin/internal/lib/service/excel/sel"
)

// FloodHourlyRepo reads observation records.
type FloodHourlyRepo interface {
	GetReservoirFloodHourlyRange(ctx context.Context, orgIDs []int64, start, end time.Time) ([]floodmodel.HourlyRecord, error)
}

// FloodConfigRepo reads the per-organization config (which orgs to include
// and in what order).
type FloodConfigRepo interface {
	GetAllReservoirFloodConfigs(ctx context.Context) ([]floodmodel.Config, error)
}

// Service builds the report.
type Service struct {
	hourly FloodHourlyRepo
	cfg    FloodConfigRepo
	loc    *time.Location
	log    *slog.Logger
}

// NewService constructs a Service.
func NewService(hourly FloodHourlyRepo, cfg FloodConfigRepo, loc *time.Location, log *slog.Logger) *Service {
	return &Service{hourly: hourly, cfg: cfg, loc: loc, log: log}
}

// BuildReport reads config + hourly data and assembles a generator-ready Report.
// `date` is the report date (only Year/Month/Day are used). `hour` is 0..23.
// `tCurr` = (date hour:00 in loc); `tPrev` = tCurr - 1h.
func (s *Service) BuildReport(ctx context.Context, date time.Time, hour int, authorShort string) (*selgen.Report, error) {
	const op = "service.sel.BuildReport"

	tCurr := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, s.loc)
	tPrev := tCurr.Add(-time.Hour)

	configs, err := s.cfg.GetAllReservoirFloodConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: configs: %w", op, err)
	}
	// Filter active and sort by SortOrder. Using a fresh allocation rather
	// than configs[:0:0] to keep intent obvious to readers.
	active := make([]floodmodel.Config, 0, len(configs))
	for _, c := range configs {
		if c.IsActive {
			active = append(active, c)
		}
	}
	sort.SliceStable(active, func(i, j int) bool {
		return active[i].SortOrder < active[j].SortOrder
	})

	if len(active) == 0 {
		return &selgen.Report{
			Date: date, Hour: hour, AuthorShort: authorShort,
		}, nil
	}

	orgIDs := make([]int64, 0, len(active))
	for _, c := range active {
		orgIDs = append(orgIDs, c.OrganizationID)
	}

	// Half-open window covering both target points with a small tail buffer.
	records, err := s.hourly.GetReservoirFloodHourlyRange(ctx, orgIDs, tPrev.UTC(), tCurr.Add(time.Hour).UTC())
	if err != nil {
		return nil, fmt.Errorf("%s: hourly range: %w", op, err)
	}

	byOrg := make(map[int64][]floodmodel.HourlyRecord, len(active))
	for _, rec := range records {
		byOrg[rec.OrganizationID] = append(byOrg[rec.OrganizationID], rec)
	}

	rows := make([]selgen.ReservoirRow, 0, len(active))
	for _, cfg := range active {
		rows = append(rows, s.buildRow(cfg, byOrg[cfg.OrganizationID], tPrev, tCurr))
	}

	return &selgen.Report{
		Date:        date,
		Hour:        hour,
		AuthorShort: authorShort,
		Reservoirs:  rows,
	}, nil
}

// buildRow folds the org's records into one ReservoirRow. duty_name takes
// the current-hour value; if absent, falls back to the previous-hour value.
// Other fields use their respective hour's record only (no fallback).
func (s *Service) buildRow(cfg floodmodel.Config, recs []floodmodel.HourlyRecord, tPrev, tCurr time.Time) selgen.ReservoirRow {
	row := selgen.ReservoirRow{Name: cfg.OrganizationName}

	var prev, curr *floodmodel.HourlyRecord
	for i := range recs {
		local := recs[i].RecordedAt.In(s.loc)
		switch {
		case local.Equal(tPrev):
			prev = &recs[i]
		case local.Equal(tCurr):
			curr = &recs[i]
		}
	}

	if prev != nil {
		row.LevelPrev = prev.WaterLevelM
		row.VolumePrev = prev.WaterVolumeMlnM3
		row.InflowPrev = prev.InflowM3s
		row.OutflowPrev = prev.OutflowM3s
		row.GESFlowPrev = prev.GESFlowM3s
		row.CapacityPrev = prev.CapacityMwt
		row.IdleDischargePrev = prev.IdleDischargeM3s
	}
	if curr != nil {
		row.LevelCurr = curr.WaterLevelM
		row.VolumeCurr = curr.WaterVolumeMlnM3
		row.InflowCurr = curr.InflowM3s
		row.OutflowCurr = curr.OutflowM3s
		row.GESFlowCurr = curr.GESFlowM3s
		row.CapacityCurr = curr.CapacityMwt
		row.IdleDischargeCurr = curr.IdleDischargeM3s
		if curr.WeatherCondition != nil {
			row.WeatherCondition = *curr.WeatherCondition
		}
		row.TemperatureC = curr.TemperatureC
		if curr.DutyName != nil {
			row.DutyName = *curr.DutyName
		}
	}
	if row.DutyName == "" && prev != nil && prev.DutyName != nil {
		row.DutyName = *prev.DutyName
	}
	return row
}
