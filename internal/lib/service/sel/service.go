// Package sel transforms reservoir-flood-hourly records into a sel-generator
// Report.
//
// Selection semantics (see docs/plans/feature-reservoir-flood-prev-flex.md):
//   - curr = exact-match record at tCurr (date+hour in loc). If absent →
//     curr-side cells render as "-".
//   - prev = most recent record with recorded_at < tCurr per organization.
//     No upper bound on age — observations may be hourly at night, every
//     three hours during day, with skips; the report follows whatever data
//     the operators actually entered.
//
// Two independent repo calls drive this: GetReservoirFloodHourlyRange for
// the strict curr window, GetReservoirFloodHourlyLatestBefore for one row
// per org. The windows don't overlap, so no transaction is required to keep
// a single row internally consistent.
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

// FloodHourlyRepo reads observation records by time window (curr) and by
// "latest before T per org" (prev). Both methods live on the same Postgres
// repo in production, so this single interface keeps the Wire wiring trivial.
type FloodHourlyRepo interface {
	GetReservoirFloodHourlyRange(ctx context.Context, orgIDs []int64, start, end time.Time) ([]floodmodel.HourlyRecord, error)
	GetReservoirFloodHourlyLatestBefore(ctx context.Context, orgIDs []int64, before time.Time) ([]floodmodel.HourlyRecord, error)
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
// `tCurr` = (date hour:00 in loc); prev is whatever the repo returns for
// "latest record per org with recorded_at < tCurr".
func (s *Service) BuildReport(ctx context.Context, date time.Time, hour int, authorShort string) (*selgen.Report, error) {
	const op = "service.sel.BuildReport"

	tCurr := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, s.loc)

	configs, err := s.cfg.GetAllReservoirFloodConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: configs: %w", op, err)
	}
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

	currRecords, err := s.hourly.GetReservoirFloodHourlyRange(ctx, orgIDs, tCurr.UTC(), tCurr.Add(time.Hour).UTC())
	if err != nil {
		return nil, fmt.Errorf("%s: curr range: %w", op, err)
	}
	prevRecords, err := s.hourly.GetReservoirFloodHourlyLatestBefore(ctx, orgIDs, tCurr.UTC())
	if err != nil {
		return nil, fmt.Errorf("%s: prev latest-before: %w", op, err)
	}

	currByOrg := make(map[int64]*floodmodel.HourlyRecord, len(currRecords))
	for i := range currRecords {
		currByOrg[currRecords[i].OrganizationID] = &currRecords[i]
	}
	prevByOrg := make(map[int64]*floodmodel.HourlyRecord, len(prevRecords))
	for i := range prevRecords {
		prevByOrg[prevRecords[i].OrganizationID] = &prevRecords[i]
	}

	rows := make([]selgen.ReservoirRow, 0, len(active))
	for _, cfg := range active {
		rows = append(rows, s.buildRow(cfg, prevByOrg[cfg.OrganizationID], currByOrg[cfg.OrganizationID]))
	}

	return &selgen.Report{
		Date:        date,
		Hour:        hour,
		AuthorShort: authorShort,
		Reservoirs:  rows,
	}, nil
}

// buildRow folds prev/curr records into one ReservoirRow.
//
// duty_name takes the current-hour value; if absent, falls back to prev.
// This fallback may surface a months-old duty name when curr is empty —
// accepted as "show last known operator rather than blank" per business
// requirement. Documented in docs/sel-export.md.
//
// Other fields take their hour's value directly; no cross-hour fallback.
func (s *Service) buildRow(cfg floodmodel.Config, prev, curr *floodmodel.HourlyRecord) selgen.ReservoirRow {
	row := selgen.ReservoirRow{Name: cfg.OrganizationName}

	if prev != nil {
		row.LevelPrev = prev.WaterLevelM
		row.VolumePrev = prev.WaterVolumeMlnM3
		row.InflowPrev = prev.InflowM3s
		row.OutflowPrev = prev.OutflowM3s
		row.GESFlowPrev = prev.GESFlowM3s
		row.CapacityPrev = prev.CapacityMwt
		row.IdleDischargePrev = prev.IdleDischargeM3s

		at := prev.RecordedAt.In(s.loc)
		row.PrevAt = &at
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
