package discharge

import (
	"context"
	"errors"
	"srmt-admin/internal/storage"
	"testing"
	"time"
)

// mockRepository is a mock implementation of Repository for testing.
type mockRepository struct {
	checkOngoingFunc func(ctx context.Context, orgID int64) (int64, bool, error)
	closeFunc        func(ctx context.Context, id int64, endTime time.Time) error
}

func (m *mockRepository) CheckOngoingDischarge(ctx context.Context, orgID int64) (int64, bool, error) {
	return m.checkOngoingFunc(ctx, orgID)
}

func (m *mockRepository) CloseDischarge(ctx context.Context, id int64, endTime time.Time) error {
	return m.closeFunc(ctx, id, endTime)
}

func TestEnsureNoOngoingDischarge_NoOngoing(t *testing.T) {
	svc := NewService(&mockRepository{
		checkOngoingFunc: func(_ context.Context, _ int64) (int64, bool, error) {
			return 0, false, nil
		},
	})

	err := svc.EnsureNoOngoingDischarge(context.Background(), 1, false, time.Now())
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestEnsureNoOngoingDischarge_ExistsNoForce(t *testing.T) {
	svc := NewService(&mockRepository{
		checkOngoingFunc: func(_ context.Context, _ int64) (int64, bool, error) {
			return 42, true, nil
		},
	})

	err := svc.EnsureNoOngoingDischarge(context.Background(), 1, false, time.Now())
	if !errors.Is(err, storage.ErrOngoingDischargeExists) {
		t.Fatalf("expected ErrOngoingDischargeExists, got %v", err)
	}
}

func TestEnsureNoOngoingDischarge_ExistsForce(t *testing.T) {
	var closedID int64
	svc := NewService(&mockRepository{
		checkOngoingFunc: func(_ context.Context, _ int64) (int64, bool, error) {
			return 42, true, nil
		},
		closeFunc: func(_ context.Context, id int64, _ time.Time) error {
			closedID = id
			return nil
		},
	})

	err := svc.EnsureNoOngoingDischarge(context.Background(), 1, true, time.Now())
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if closedID != 42 {
		t.Fatalf("expected close to be called with id 42, got %d", closedID)
	}
}

func TestEnsureNoOngoingDischarge_CheckError(t *testing.T) {
	repoErr := errors.New("db connection failed")
	svc := NewService(&mockRepository{
		checkOngoingFunc: func(_ context.Context, _ int64) (int64, bool, error) {
			return 0, false, repoErr
		},
	})

	err := svc.EnsureNoOngoingDischarge(context.Background(), 1, false, time.Now())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestEnsureNoOngoingDischarge_CloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	svc := NewService(&mockRepository{
		checkOngoingFunc: func(_ context.Context, _ int64) (int64, bool, error) {
			return 42, true, nil
		},
		closeFunc: func(_ context.Context, _ int64, _ time.Time) error {
			return closeErr
		},
	})

	err := svc.EnsureNoOngoingDischarge(context.Background(), 1, true, time.Now())
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected close error, got %v", err)
	}
}

func TestEnsureNoOngoingDischarge_ForceCloseCheckConstraint(t *testing.T) {
	// Simulate: existing discharge starts at 21:00, new one at 05:00.
	// CloseDischarge sets end_time=05:00 which violates CHECK (end_time > start_time).
	// The repo returns ErrCheckConstraintViolation, service should translate to ErrDischargeEndBeforeStart.
	svc := NewService(&mockRepository{
		checkOngoingFunc: func(_ context.Context, _ int64) (int64, bool, error) {
			return 42, true, nil
		},
		closeFunc: func(_ context.Context, _ int64, _ time.Time) error {
			return storage.ErrCheckConstraintViolation
		},
	})

	err := svc.EnsureNoOngoingDischarge(context.Background(), 1, true, time.Now())
	if !errors.Is(err, storage.ErrDischargeEndBeforeStart) {
		t.Fatalf("expected ErrDischargeEndBeforeStart, got %v", err)
	}
}
