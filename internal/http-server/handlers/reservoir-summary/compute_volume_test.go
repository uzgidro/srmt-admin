package reservoirsummary

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

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
