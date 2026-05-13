package sel

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	floodmodel "srmt-admin/internal/lib/model/reservoir-flood"
)

// ---------- mocks ----------

// fakeRepo implements both FloodHourlyRepo (range) and FloodHourlyLatestRepo
// (latest-before-T) — needed because BuildReport now issues two independent
// queries: one for the exact curr-hour window, one for the latest record per
// org strictly before tCurr.
type fakeRepo struct {
	// Range query — used for curr only.
	rangeOut  []floodmodel.HourlyRecord
	rangeErr  error
	gotRangeIDs  []int64
	gotRangeFrom time.Time
	gotRangeTo   time.Time

	// LatestBefore query — used for prev (≤1 record per org).
	latestOut []floodmodel.HourlyRecord
	latestErr error
	gotLatestIDs    []int64
	gotLatestBefore time.Time
}

func (f *fakeRepo) GetReservoirFloodHourlyRange(_ context.Context, ids []int64, from, to time.Time) ([]floodmodel.HourlyRecord, error) {
	f.gotRangeIDs = ids
	f.gotRangeFrom = from
	f.gotRangeTo = to
	return f.rangeOut, f.rangeErr
}

func (f *fakeRepo) GetReservoirFloodHourlyLatestBefore(_ context.Context, ids []int64, before time.Time) ([]floodmodel.HourlyRecord, error) {
	f.gotLatestIDs = ids
	f.gotLatestBefore = before
	return f.latestOut, f.latestErr
}

type fakeConfig struct {
	out []floodmodel.Config
	err error
}

func (f *fakeConfig) GetAllReservoirFloodConfigs(_ context.Context) ([]floodmodel.Config, error) {
	return f.out, f.err
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func tashkent(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return loc
}

func ptr(v float64) *float64 { return &v }
func sptr(s string) *string  { return &s }

// ---------- tests ----------

func TestBuildReport_EmptyConfig(t *testing.T) {
	loc := tashkent(t)
	svc := NewService(&fakeRepo{}, &fakeConfig{out: nil}, loc, discardLogger())
	r, err := svc.BuildReport(context.Background(), time.Date(2026, 5, 4, 0, 0, 0, 0, loc), 0, "И. Иванов")
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if len(r.Reservoirs) != 0 {
		t.Errorf("want 0 reservoirs, got %d", len(r.Reservoirs))
	}
	if r.AuthorShort != "И. Иванов" {
		t.Errorf("AuthorShort: want И. Иванов, got %q", r.AuthorShort)
	}
}

func TestBuildReport_BothPoints_Hour0(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 0, 0, 0, 0, loc)
	tPrev := tCurr.Add(-time.Hour)
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(929.64), WaterVolumeMlnM3: ptr(6.12), InflowM3s: ptr(205), DutyName: sptr("Петров П.")},
		},
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 96, RecordedAt: tPrev, WaterLevelM: ptr(929.57), WaterVolumeMlnM3: ptr(6.091), InflowM3s: ptr(205), DutyName: sptr("Иванов И.")},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{
		{OrganizationID: 96, OrganizationName: "Чотқол сув омбори", SortOrder: 1, IsActive: true},
	}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, err := svc.BuildReport(context.Background(), tCurr, 0, "И. Иванов")
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if len(r.Reservoirs) != 1 {
		t.Fatalf("want 1 reservoir, got %d", len(r.Reservoirs))
	}
	row := r.Reservoirs[0]
	if row.Name != "Чотқол сув омбори" {
		t.Errorf("Name verbatim: want %q, got %q", "Чотқол сув омбори", row.Name)
	}
	if row.LevelPrev == nil || *row.LevelPrev != 929.57 {
		t.Errorf("LevelPrev: want 929.57, got %v", row.LevelPrev)
	}
	if row.LevelCurr == nil || *row.LevelCurr != 929.64 {
		t.Errorf("LevelCurr: want 929.64, got %v", row.LevelCurr)
	}
	if row.DutyName != "Петров П." {
		t.Errorf("DutyName from current: want Петров П., got %q", row.DutyName)
	}
	if row.PrevAt == nil || !row.PrevAt.Equal(tPrev) {
		t.Errorf("PrevAt: want %v, got %v", tPrev, row.PrevAt)
	}

	// Range query targets [tCurr, tCurr+1h).
	if !repo.gotRangeFrom.Equal(tCurr.UTC()) {
		t.Errorf("range.from: want %v, got %v", tCurr.UTC(), repo.gotRangeFrom)
	}
	if !repo.gotRangeTo.Equal(tCurr.Add(time.Hour).UTC()) {
		t.Errorf("range.to: want %v, got %v", tCurr.Add(time.Hour).UTC(), repo.gotRangeTo)
	}
	if !repo.gotLatestBefore.Equal(tCurr.UTC()) {
		t.Errorf("latest.before: want %v, got %v", tCurr.UTC(), repo.gotLatestBefore)
	}
}

func TestBuildReport_BothPoints_HourMid(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 15, 0, 0, 0, loc)
	tPrev := time.Date(2026, 5, 4, 14, 0, 0, 0, loc)
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(810)},
		},
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 96, RecordedAt: tPrev, WaterLevelM: ptr(800)},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 96, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, err := svc.BuildReport(context.Background(), time.Date(2026, 5, 4, 0, 0, 0, 0, loc), 15, "I. I.")
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	row := r.Reservoirs[0]
	if row.LevelPrev == nil || *row.LevelPrev != 800 {
		t.Errorf("LevelPrev: want 800, got %v", row.LevelPrev)
	}
	if row.LevelCurr == nil || *row.LevelCurr != 810 {
		t.Errorf("LevelCurr: want 810, got %v", row.LevelCurr)
	}
}

func TestBuildReport_OnlyCurrent(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 0, 0, 0, 0, loc)
	repo := &fakeRepo{rangeOut: []floodmodel.HourlyRecord{
		{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(929.64)},
	}}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 96, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), tCurr, 0, "")
	row := r.Reservoirs[0]
	if row.LevelPrev != nil {
		t.Errorf("LevelPrev: want nil, got %v", row.LevelPrev)
	}
	if row.LevelCurr == nil || *row.LevelCurr != 929.64 {
		t.Errorf("LevelCurr: want 929.64, got %v", row.LevelCurr)
	}
	if row.PrevAt != nil {
		t.Errorf("PrevAt: want nil when no prev, got %v", row.PrevAt)
	}
}

func TestBuildReport_DutyFallbackToPrev(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 0, 0, 0, 0, loc)
	tPrev := tCurr.Add(-time.Hour)
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			// curr record exists but has no duty_name.
			{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(100)},
		},
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 96, RecordedAt: tPrev, DutyName: sptr("Иванов И.")},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 96, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), tCurr, 0, "")
	if r.Reservoirs[0].DutyName != "Иванов И." {
		t.Errorf("DutyName fallback: want Иванов И., got %q", r.Reservoirs[0].DutyName)
	}
}

func TestBuildReport_OrderingFromConfig(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 0, 0, 0, 0, loc)
	cfg := &fakeConfig{out: []floodmodel.Config{
		{OrganizationID: 100, OrganizationName: "Third", SortOrder: 3, IsActive: true},
		{OrganizationID: 200, OrganizationName: "First", SortOrder: 1, IsActive: true},
		{OrganizationID: 300, OrganizationName: "Second", SortOrder: 2, IsActive: true},
		{OrganizationID: 400, OrganizationName: "Inactive", SortOrder: 0, IsActive: false},
	}}
	repo := &fakeRepo{rangeOut: []floodmodel.HourlyRecord{
		{OrganizationID: 100, RecordedAt: tCurr},
		{OrganizationID: 200, RecordedAt: tCurr},
		{OrganizationID: 300, RecordedAt: tCurr},
	}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), tCurr, 0, "")
	if len(r.Reservoirs) != 3 {
		t.Fatalf("want 3 reservoirs (inactive filtered), got %d", len(r.Reservoirs))
	}
	wantOrder := []string{"First", "Second", "Third"}
	for i, w := range wantOrder {
		if r.Reservoirs[i].Name != w {
			t.Errorf("Reservoirs[%d].Name: want %q, got %q", i, w, r.Reservoirs[i].Name)
		}
	}
}

func TestBuildReport_PassesReservoirNameVerbatim(t *testing.T) {
	loc := tashkent(t)
	cfg := &fakeConfig{out: []floodmodel.Config{
		{OrganizationID: 1, OrganizationName: "Чорвоқ сув омбори", SortOrder: 1, IsActive: true},
		{OrganizationID: 2, OrganizationName: "Сардоба", SortOrder: 2, IsActive: true},
		{OrganizationID: 3, OrganizationName: "", SortOrder: 3, IsActive: true},
		{OrganizationID: 4, OrganizationName: "  Андижон   сув   омбори  ", SortOrder: 4, IsActive: true},
	}}
	svc := NewService(&fakeRepo{}, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), time.Date(2026, 5, 4, 0, 0, 0, 0, loc), 0, "")
	want := []string{"Чорвоқ сув омбори", "Сардоба", "", "  Андижон   сув   омбори  "}
	for i, w := range want {
		if r.Reservoirs[i].Name != w {
			t.Errorf("Reservoirs[%d].Name: want %q, got %q", i, w, r.Reservoirs[i].Name)
		}
	}
}

func TestBuildReport_NewMetricsRoundTrip(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 0, 0, 0, 0, loc)
	tPrev := tCurr.Add(-time.Hour)
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tCurr, CapacityMwt: ptr(98.2), WeatherCondition: sptr("ясно"), TemperatureC: ptr(-3.5)},
		},
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tPrev, CapacityMwt: ptr(100.5)},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 1, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), tCurr, 0, "")
	row := r.Reservoirs[0]
	if row.CapacityPrev == nil || *row.CapacityPrev != 100.5 {
		t.Errorf("CapacityPrev: want 100.5, got %v", row.CapacityPrev)
	}
	if row.CapacityCurr == nil || *row.CapacityCurr != 98.2 {
		t.Errorf("CapacityCurr: want 98.2, got %v", row.CapacityCurr)
	}
	if row.WeatherCondition != "ясно" {
		t.Errorf("WeatherCondition: want ясно, got %q", row.WeatherCondition)
	}
	if row.TemperatureC == nil || *row.TemperatureC != -3.5 {
		t.Errorf("TemperatureC (negative valid): want -3.5, got %v", row.TemperatureC)
	}
}

// ---------- new tests: flex-prev semantics ----------

// TestBuildReport_PrevOlderThanOneHour: prev recorded 3h before tCurr.
// New semantics: pick whatever LatestBefore returns regardless of age.
func TestBuildReport_PrevOlderThanOneHour(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 13, 15, 0, 0, 0, loc)
	tPrev := time.Date(2026, 5, 13, 12, 0, 0, 0, loc) // 3h earlier
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tCurr, WaterLevelM: ptr(810)},
		},
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tPrev, WaterLevelM: ptr(800)},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 1, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), time.Date(2026, 5, 13, 0, 0, 0, 0, loc), 15, "")
	row := r.Reservoirs[0]
	if row.LevelPrev == nil || *row.LevelPrev != 800 {
		t.Errorf("LevelPrev: want 800 (from 3h-old record), got %v", row.LevelPrev)
	}
	if row.PrevAt == nil || !row.PrevAt.Equal(tPrev) {
		t.Errorf("PrevAt: want %v, got %v", tPrev, row.PrevAt)
	}
}

// TestBuildReport_PrevFromPreviousDay: tCurr=00:00, prev=23:00 yesterday.
// PrevAt must carry the previous date.
func TestBuildReport_PrevFromPreviousDay(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 13, 0, 0, 0, 0, loc)
	tPrev := time.Date(2026, 5, 12, 23, 0, 0, 0, loc)
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tCurr, WaterLevelM: ptr(810)},
		},
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tPrev, WaterLevelM: ptr(800)},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 1, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), time.Date(2026, 5, 13, 0, 0, 0, 0, loc), 0, "")
	row := r.Reservoirs[0]
	if row.PrevAt == nil {
		t.Fatal("PrevAt: want non-nil, got nil")
	}
	if !row.PrevAt.Equal(tPrev) {
		t.Errorf("PrevAt: want %v, got %v", tPrev, row.PrevAt)
	}
}

// TestBuildReport_MixedPrevTimes: three orgs with prev at 11:00, 12:00, 14:00.
// Each row carries its own PrevAt.
func TestBuildReport_MixedPrevTimes(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 13, 15, 0, 0, 0, loc)
	prev1 := time.Date(2026, 5, 13, 11, 0, 0, 0, loc)
	prev2 := time.Date(2026, 5, 13, 12, 0, 0, 0, loc)
	prev3 := time.Date(2026, 5, 13, 14, 0, 0, 0, loc)
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tCurr, WaterLevelM: ptr(100)},
			{OrganizationID: 2, RecordedAt: tCurr, WaterLevelM: ptr(200)},
			{OrganizationID: 3, RecordedAt: tCurr, WaterLevelM: ptr(300)},
		},
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: prev1, WaterLevelM: ptr(99)},
			{OrganizationID: 2, RecordedAt: prev2, WaterLevelM: ptr(199)},
			{OrganizationID: 3, RecordedAt: prev3, WaterLevelM: ptr(299)},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{
		{OrganizationID: 1, OrganizationName: "A", SortOrder: 1, IsActive: true},
		{OrganizationID: 2, OrganizationName: "B", SortOrder: 2, IsActive: true},
		{OrganizationID: 3, OrganizationName: "C", SortOrder: 3, IsActive: true},
	}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), time.Date(2026, 5, 13, 0, 0, 0, 0, loc), 15, "")
	wantPrev := []time.Time{prev1, prev2, prev3}
	for i, wp := range wantPrev {
		if r.Reservoirs[i].PrevAt == nil || !r.Reservoirs[i].PrevAt.Equal(wp) {
			t.Errorf("Reservoirs[%d].PrevAt: want %v, got %v", i, wp, r.Reservoirs[i].PrevAt)
		}
	}
}

// TestBuildReport_NoPrev: no LatestBefore record at all → PrevAt nil, prev metrics nil.
func TestBuildReport_NoPrev(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 13, 15, 0, 0, 0, loc)
	repo := &fakeRepo{
		rangeOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tCurr, WaterLevelM: ptr(810)},
		},
		latestOut: nil,
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 1, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), time.Date(2026, 5, 13, 0, 0, 0, 0, loc), 15, "")
	row := r.Reservoirs[0]
	if row.PrevAt != nil {
		t.Errorf("PrevAt: want nil, got %v", row.PrevAt)
	}
	if row.LevelPrev != nil {
		t.Errorf("LevelPrev: want nil, got %v", row.LevelPrev)
	}
}

// TestBuildReport_CurrAbsent: tCurr empty; prev still rendered.
func TestBuildReport_CurrAbsent(t *testing.T) {
	loc := tashkent(t)
	tPrev := time.Date(2026, 5, 13, 12, 0, 0, 0, loc)
	repo := &fakeRepo{
		rangeOut: nil, // no curr record
		latestOut: []floodmodel.HourlyRecord{
			{OrganizationID: 1, RecordedAt: tPrev, WaterLevelM: ptr(800)},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 1, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(repo, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), time.Date(2026, 5, 13, 0, 0, 0, 0, loc), 15, "")
	row := r.Reservoirs[0]
	if row.LevelCurr != nil {
		t.Errorf("LevelCurr: want nil (no curr), got %v", row.LevelCurr)
	}
	if row.LevelPrev == nil || *row.LevelPrev != 800 {
		t.Errorf("LevelPrev: want 800, got %v", row.LevelPrev)
	}
	if row.PrevAt == nil || !row.PrevAt.Equal(tPrev) {
		t.Errorf("PrevAt: want %v, got %v", tPrev, row.PrevAt)
	}
}
