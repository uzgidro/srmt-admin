package dayrotation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"srmt-admin/internal/storage/repo"
)

type mockRotator struct {
	result *repo.DayRotationResult
	err    error
}

func (m *mockRotator) RotateDayBoundary(ctx context.Context, cutoff time.Time) (*repo.DayRotationResult, error) {
	return m.result, m.err
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustLoadLocation(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}
	return loc
}

func TestRun_Success(t *testing.T) {
	loc := mustLoadLocation(t)
	mock := &mockRotator{
		result: &repo.DayRotationResult{
			LinkedDischargesRotated: 2,
			DischargesRotated:       3,
		},
		err: nil,
	}

	svc := NewService(mock, loc, newTestLogger())
	svc.Run(context.Background(), time.Now())
}

func TestRun_Error(t *testing.T) {
	loc := mustLoadLocation(t)
	mock := &mockRotator{
		result: nil,
		err:    errors.New("db error"),
	}

	svc := NewService(mock, loc, newTestLogger())
	svc.Run(context.Background(), time.Now())
}

func TestStartScheduler_ContextCancel(t *testing.T) {
	loc := mustLoadLocation(t)
	mock := &mockRotator{
		result: &repo.DayRotationResult{},
		err:    nil,
	}

	svc := NewService(mock, loc, newTestLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		svc.StartScheduler(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// goroutine exited as expected
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler goroutine did not exit after context cancellation")
	}
}
