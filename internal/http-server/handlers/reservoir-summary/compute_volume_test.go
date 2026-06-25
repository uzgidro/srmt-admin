package reservoirsummary

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"srmt-admin/internal/lib/dto"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/storage"
)

type mockCurveRepo struct {
	calls   []curveCall
	volume  float64
	err     error
}

type curveCall struct {
	orgID int64
	level float64
}

func (m *mockCurveRepo) GetVolumeByLevelByOrg(_ context.Context, orgID int64, level float64) (float64, error) {
	m.calls = append(m.calls, curveCall{orgID: orgID, level: level})
	return m.volume, m.err
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func TestComputeVolumeFromLevel_OK(t *testing.T) {
	repo := &mockCurveRepo{volume: 100.0}
	got, ok := computeVolumeFromLevel(context.Background(), newTestLogger(), repo, 96, 200.5)
	if !ok {
		t.Fatalf("expected ok=true, got false")
	}
	if got != 100.0 {
		t.Errorf("expected 100.0, got %f", got)
	}
	if len(repo.calls) != 1 || repo.calls[0].orgID != 96 || repo.calls[0].level != 200.5 {
		t.Errorf("unexpected calls: %+v", repo.calls)
	}
}

func TestComputeVolumeFromLevel_NotConfigured(t *testing.T) {
	repo := &mockCurveRepo{err: storage.ErrLevelVolumeNotConfigured}
	got, ok := computeVolumeFromLevel(context.Background(), newTestLogger(), repo, 96, 200.0)
	if ok {
		t.Errorf("expected ok=false on not-configured, got true")
	}
	if got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestComputeVolumeFromLevel_OutOfRange(t *testing.T) {
	repo := &mockCurveRepo{err: storage.ErrLevelOutOfCurveRange}
	got, ok := computeVolumeFromLevel(context.Background(), newTestLogger(), repo, 96, 999.0)
	if ok {
		t.Errorf("expected ok=false on out-of-range, got true")
	}
	if got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

func TestComputeVolumeFromLevel_OtherError(t *testing.T) {
	repo := &mockCurveRepo{err: errors.New("db down")}
	got, ok := computeVolumeFromLevel(context.Background(), newTestLogger(), repo, 96, 200.0)
	if ok {
		t.Errorf("expected ok=false on generic error, got true")
	}
	if got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

// SNAPSHOT WINS — common precondition for BOTH strategies.
// Operators reported that any manual POST into reservoir_data.volume_mln_m3
// must surface in the report regardless of volume_source. The strategy
// switch only fires when the snapshot is zero (i.e. operator hasn't typed
// anything for that day yet).
func TestApplyStaticFallbacks_LevelVolume_SnapshotWinsWhenPresent(t *testing.T) {
	orgID := int64(42)
	staticVolume := 999.0
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 100}, // snapshot from operator
	}}
	dayBegin := map[int64]*dto.OrganizationWithData{
		orgID: {Data: &dto.ReservoirData{Volume: &staticVolume}},
	}
	curve := &mockCurveRepo{volume: 150}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "level_volume"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, configs)

	if summaries[0].Volume.Current != 100 {
		t.Errorf("Volume.Current: want 100 (snapshot wins), got %v", summaries[0].Volume.Current)
	}
	if len(curve.calls) != 0 {
		t.Errorf("curve must not be queried when snapshot is non-zero; got %+v", curve.calls)
	}
}

func TestApplyStaticFallbacks_Static_SnapshotWinsWhenPresent(t *testing.T) {
	orgID := int64(42)
	staticVolume := 999.0
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 100},
	}}
	dayBegin := map[int64]*dto.OrganizationWithData{
		orgID: {Data: &dto.ReservoirData{Volume: &staticVolume}},
	}
	curve := &mockCurveRepo{volume: 150}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "static"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, configs)

	if summaries[0].Volume.Current != 100 {
		t.Errorf("Volume.Current: want 100 (snapshot wins), got %v", summaries[0].Volume.Current)
	}
}

// volume_source = "static" with zero snapshot: static.uz wins over curve.
// This is the bug fix — previously the curve was always preferred and
// static.uz was the last-resort fallback, which inverted the operator's
// explicit "static" choice.
func TestApplyStaticFallbacks_Static_StaticUzBeatsCurveWhenSnapshotZero(t *testing.T) {
	orgID := int64(42)
	staticVolume := 999.0
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 0},
	}}
	dayBegin := map[int64]*dto.OrganizationWithData{
		orgID: {Data: &dto.ReservoirData{Volume: &staticVolume}},
	}
	curve := &mockCurveRepo{volume: 150} // curve has an answer; must be ignored
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "static"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, configs)

	if summaries[0].Volume.Current != staticVolume {
		t.Errorf("Volume.Current: want %v (static.uz wins under static), got %v", staticVolume, summaries[0].Volume.Current)
	}
	if summaries[0].Volume.IsEdited == nil || !*summaries[0].Volume.IsEdited {
		t.Errorf("Volume.IsEdited: want true, got %v", summaries[0].Volume.IsEdited)
	}
}

// "static" + zero snapshot + no static.uz volume → curve is the fallback.
// Confirms the chain static.uz → curve → 0, not static.uz alone.
func TestApplyStaticFallbacks_Static_CurveFallbackWhenStaticUzAbsent(t *testing.T) {
	orgID := int64(42)
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 0},
	}}
	dayBegin := map[int64]*dto.OrganizationWithData{} // no static.uz entry
	curve := &mockCurveRepo{volume: 150}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "static"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, configs)

	if summaries[0].Volume.Current != 150 {
		t.Errorf("Volume.Current: want 150 (curve fallback after static.uz miss), got %v", summaries[0].Volume.Current)
	}
}

// "static" + zero snapshot + no static.uz + no curve → stays 0. This is the
// terminal degraded state; the cell must not be filled with garbage.
func TestApplyStaticFallbacks_Static_StaysZeroWhenAllSourcesEmpty(t *testing.T) {
	orgID := int64(42)
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 0},
	}}
	curve := &mockCurveRepo{err: storage.ErrLevelVolumeNotConfigured}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "static"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, nil, curve, configs)

	if summaries[0].Volume.Current != 0 {
		t.Errorf("Volume.Current: want 0 (no sources, no fallback), got %v", summaries[0].Volume.Current)
	}
}

// volume_source = "level_volume" + zero snapshot: curve wins; fallback to
// static.uz only when curve isn't configured.
func TestApplyStaticFallbacks_LevelVolume_CurveWinsWhenSnapshotZero(t *testing.T) {
	orgID := int64(42)
	staticVolume := 999.0
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 0},
	}}
	dayBegin := map[int64]*dto.OrganizationWithData{
		orgID: {Data: &dto.ReservoirData{Volume: &staticVolume}},
	}
	curve := &mockCurveRepo{volume: 150}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "level_volume"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, configs)

	if summaries[0].Volume.Current != 150 {
		t.Errorf("Volume.Current: want 150 (curve wins under level_volume), got %v", summaries[0].Volume.Current)
	}
}

func TestApplyStaticFallbacks_LevelVolume_StaticUzFallbackWhenCurveMissing(t *testing.T) {
	orgID := int64(42)
	staticVolume := 999.0
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 0},
	}}
	dayBegin := map[int64]*dto.OrganizationWithData{
		orgID: {Data: &dto.ReservoirData{Volume: &staticVolume}},
	}
	curve := &mockCurveRepo{err: storage.ErrLevelVolumeNotConfigured}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "level_volume"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, configs)

	if summaries[0].Volume.Current != staticVolume {
		t.Errorf("Volume.Current: want %v (static.uz fallback after curve miss), got %v", staticVolume, summaries[0].Volume.Current)
	}
}

func TestApplyStaticFallbacks_LevelVolume_StaysZeroWhenAllSourcesEmpty(t *testing.T) {
	orgID := int64(42)
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 0},
	}}
	curve := &mockCurveRepo{err: storage.ErrLevelVolumeNotConfigured}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "level_volume"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, nil, curve, configs)

	if summaries[0].Volume.Current != 0 {
		t.Errorf("Volume.Current: want 0 (no sources), got %v", summaries[0].Volume.Current)
	}
}

// Empty MapConfigLookup → degraded "static" default for every org. The
// regression guard from the prep PR has to track the new semantics:
// snapshot wins, then static.uz, then curve, then 0.
func TestApplyStaticFallbacks_IgnoresConfigsParam(t *testing.T) {
	orgID := int64(42)
	level := 200.5
	staticIncome := 11.0
	staticRelease := 22.0
	staticLevel := 88.0
	staticVolume := 999.0
	curveVolume := 555.0

	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: level},
	}}
	dayBegin := map[int64]*dto.OrganizationWithData{
		orgID: {Data: &dto.ReservoirData{
			AvgIncome:  &staticIncome,
			AvgRelease: &staticRelease,
			Level:      &staticLevel,
			Volume:     &staticVolume,
		}},
	}
	curve := &mockCurveRepo{volume: curveVolume}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, MapConfigLookup{})

	got := summaries[0]
	if got.Income.Current != staticIncome {
		t.Errorf("Income.Current: want %v, got %v", staticIncome, got.Income.Current)
	}
	if got.Release.Current != staticRelease {
		t.Errorf("Release.Current: want %v, got %v", staticRelease, got.Release.Current)
	}
	if got.Level.Current != level {
		t.Errorf("Level.Current: want %v (untouched), got %v", level, got.Level.Current)
	}
	// Default strategy = "static". Volume was zero → static.uz wins over curve.
	if got.Volume.Current != staticVolume {
		t.Errorf("Volume.Current: want %v (static.uz under default static), got %v", staticVolume, got.Volume.Current)
	}
}
