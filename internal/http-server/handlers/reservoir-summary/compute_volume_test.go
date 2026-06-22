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
