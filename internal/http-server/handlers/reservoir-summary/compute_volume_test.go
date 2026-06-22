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

// volume_source = "level_volume" inverts the legacy strategy: even when the
// DB snapshot is non-zero, the curve result wins. This is how operators opt
// reservoirs whose stored volume disagrees with the calibrated curve onto a
// single source of truth.
func TestApplyStaticFallbacks_LevelVolumeOverridesSnapshot(t *testing.T) {
	orgID := int64(42)
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 100}, // snapshot from DB
	}}
	curve := &mockCurveRepo{volume: 150} // curve disagrees with snapshot
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "level_volume"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, nil, curve, configs)

	if summaries[0].Volume.Current != 150 {
		t.Errorf("Volume.Current: want 150 (curve wins), got %v", summaries[0].Volume.Current)
	}
	if summaries[0].Volume.IsEdited == nil || !*summaries[0].Volume.IsEdited {
		t.Errorf("Volume.IsEdited: want true, got %v", summaries[0].Volume.IsEdited)
	}
}

// "level_volume" without a configured curve falls back to the snapshot so a
// half-finished migration (config flipped, curve not yet loaded) doesn't blank
// the value on the dashboard.
func TestApplyStaticFallbacks_LevelVolumeFallsBackToSnapshotIfCurveMissing(t *testing.T) {
	orgID := int64(42)
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 100},
	}}
	curve := &mockCurveRepo{err: storage.ErrLevelVolumeNotConfigured}
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "level_volume"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, nil, curve, configs)

	if summaries[0].Volume.Current != 100 {
		t.Errorf("Volume.Current: want 100 (snapshot preserved), got %v", summaries[0].Volume.Current)
	}
	if summaries[0].Volume.IsEdited != nil {
		t.Errorf("Volume.IsEdited: want nil (untouched), got %v", *summaries[0].Volume.IsEdited)
	}
}

// "static" matches the pre-feature behaviour exactly: a non-zero snapshot
// short-circuits any recompute, regardless of what the curve would return.
func TestApplyStaticFallbacks_StaticPreservesLegacyBehaviour(t *testing.T) {
	orgID := int64(42)
	summaries := []*reservoirsummary.ResponseModel{{
		OrganizationID: &orgID,
		Level:          reservoirsummary.ValueResponse{Current: 200},
		Volume:         reservoirsummary.ValueResponse{Current: 100},
	}}
	curve := &mockCurveRepo{volume: 999} // would be applied if the strategy were level_volume
	configs := MapConfigLookup{
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "static"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, nil, curve, configs)

	if summaries[0].Volume.Current != 100 {
		t.Errorf("Volume.Current: want 100 (snapshot, no recompute), got %v", summaries[0].Volume.Current)
	}
	if len(curve.calls) != 0 {
		t.Errorf("curve must not be called when snapshot is non-zero under static; got %+v", curve.calls)
	}
}

// Even under "static", a zero snapshot still triggers the curve recompute —
// and the curve wins over the static.uz volume fallback. Pins that the
// pre-existing snapshot→curve→static.uz priority is preserved in this branch.
func TestApplyStaticFallbacks_StaticUsesCurveOnlyWhenSnapshotZero(t *testing.T) {
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
		orgID: reservoirsummary.ReservoirSummaryConfig{OrganizationID: orgID, VolumeSource: "static"},
	}

	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, configs)

	if summaries[0].Volume.Current != 150 {
		t.Errorf("Volume.Current: want 150 (curve beats static.uz), got %v", summaries[0].Volume.Current)
	}
}

// MapConfigLookup is introduced as a no-op seam ahead of the modsnow_enabled /
// volume_source features. This test pins the contract: passing an empty
// ConfigLookup must not change applyStaticFallbacks' existing behaviour —
// snapshot→curve→static.uz priority still holds, no panic on missing keys.
// Both feature branches (Module A / Module B) build on top of this seam, so a
// regression here would silently break either of them.
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

	// Empty MapConfigLookup must be treated identically to "no config" — i.e.
	// the legacy behaviour is preserved bit-for-bit.
	applyStaticFallbacks(context.Background(), newTestLogger(), summaries, dayBegin, curve, MapConfigLookup{})

	got := summaries[0]
	if got.Income.Current != staticIncome {
		t.Errorf("Income.Current: want %v, got %v", staticIncome, got.Income.Current)
	}
	if got.Release.Current != staticRelease {
		t.Errorf("Release.Current: want %v, got %v", staticRelease, got.Release.Current)
	}
	// Level was non-zero in the summary already, so the static.uz Level must NOT overwrite it.
	if got.Level.Current != level {
		t.Errorf("Level.Current: want %v (untouched), got %v", level, got.Level.Current)
	}
	// Volume was zero → curve wins over static.uz volume.
	if got.Volume.Current != curveVolume {
		t.Errorf("Volume.Current: want %v (curve), got %v", curveVolume, got.Volume.Current)
	}
}
