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

type fakeHourly struct {
	want    []int64
	gotIDs  []int64
	gotFrom time.Time
	gotTo   time.Time
	out     []floodmodel.HourlyRecord
	err     error
}

func (f *fakeHourly) GetReservoirFloodHourlyRange(_ context.Context, ids []int64, from, to time.Time) ([]floodmodel.HourlyRecord, error) {
	f.gotIDs = ids
	f.gotFrom = from
	f.gotTo = to
	return f.out, f.err
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
	svc := NewService(&fakeHourly{}, &fakeConfig{out: nil}, loc, discardLogger())
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
	hr := &fakeHourly{
		out: []floodmodel.HourlyRecord{
			{OrganizationID: 96, RecordedAt: tPrev, WaterLevelM: ptr(929.57), WaterVolumeMlnM3: ptr(6.091), InflowM3s: ptr(205), DutyName: sptr("Иванов И.")},
			{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(929.64), WaterVolumeMlnM3: ptr(6.12), InflowM3s: ptr(205), DutyName: sptr("Петров П.")},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{
		{OrganizationID: 96, OrganizationName: "Чотқол сув омбори", SortOrder: 1, IsActive: true},
	}}
	svc := NewService(hr, cfg, loc, discardLogger())
	r, err := svc.BuildReport(context.Background(), tCurr, 0, "И. Иванов")
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if len(r.Reservoirs) != 1 {
		t.Fatalf("want 1 reservoir, got %d", len(r.Reservoirs))
	}
	row := r.Reservoirs[0]
	if row.Name != "Чотқол" {
		t.Errorf("Name shortened: want Чотқол, got %q", row.Name)
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
}

func TestBuildReport_BothPoints_HourMid(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 15, 0, 0, 0, loc)
	tPrev := time.Date(2026, 5, 4, 14, 0, 0, 0, loc)
	hr := &fakeHourly{
		out: []floodmodel.HourlyRecord{
			{OrganizationID: 96, RecordedAt: tPrev, WaterLevelM: ptr(800)},
			{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(810)},
		},
	}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 96, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(hr, cfg, loc, discardLogger())
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
	hr := &fakeHourly{out: []floodmodel.HourlyRecord{
		{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(929.64)},
	}}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 96, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(hr, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), tCurr, 0, "")
	row := r.Reservoirs[0]
	if row.LevelPrev != nil {
		t.Errorf("LevelPrev: want nil, got %v", row.LevelPrev)
	}
	if row.LevelCurr == nil || *row.LevelCurr != 929.64 {
		t.Errorf("LevelCurr: want 929.64, got %v", row.LevelCurr)
	}
}

func TestBuildReport_DutyFallbackToPrev(t *testing.T) {
	loc := tashkent(t)
	tCurr := time.Date(2026, 5, 4, 0, 0, 0, 0, loc)
	tPrev := tCurr.Add(-time.Hour)
	hr := &fakeHourly{out: []floodmodel.HourlyRecord{
		{OrganizationID: 96, RecordedAt: tPrev, DutyName: sptr("Иванов И.")},
		// curr record exists but has no duty_name.
		{OrganizationID: 96, RecordedAt: tCurr, WaterLevelM: ptr(100)},
	}}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 96, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(hr, cfg, loc, discardLogger())
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
	hr := &fakeHourly{out: []floodmodel.HourlyRecord{
		{OrganizationID: 100, RecordedAt: tCurr},
		{OrganizationID: 200, RecordedAt: tCurr},
		{OrganizationID: 300, RecordedAt: tCurr},
	}}
	svc := NewService(hr, cfg, loc, discardLogger())
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

func TestBuildReport_ShortensReservoirName(t *testing.T) {
	loc := tashkent(t)
	cfg := &fakeConfig{out: []floodmodel.Config{
		{OrganizationID: 1, OrganizationName: "Чорвоқ сув омбори", SortOrder: 1, IsActive: true},
		{OrganizationID: 2, OrganizationName: "Сардоба", SortOrder: 2, IsActive: true},
		{OrganizationID: 3, OrganizationName: "", SortOrder: 3, IsActive: true},
		{OrganizationID: 4, OrganizationName: "  Андижон   сув   омбори  ", SortOrder: 4, IsActive: true},
	}}
	svc := NewService(&fakeHourly{}, cfg, loc, discardLogger())
	r, _ := svc.BuildReport(context.Background(), time.Date(2026, 5, 4, 0, 0, 0, 0, loc), 0, "")
	want := []string{"Чорвоқ", "Сардоба", "", "Андижон"}
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
	hr := &fakeHourly{out: []floodmodel.HourlyRecord{
		{OrganizationID: 1, RecordedAt: tPrev, CapacityMwt: ptr(100.5)},
		{OrganizationID: 1, RecordedAt: tCurr, CapacityMwt: ptr(98.2), WeatherCondition: sptr("ясно"), TemperatureC: ptr(-3.5)},
	}}
	cfg := &fakeConfig{out: []floodmodel.Config{{OrganizationID: 1, OrganizationName: "X", SortOrder: 1, IsActive: true}}}
	svc := NewService(hr, cfg, loc, discardLogger())
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
