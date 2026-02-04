package alarm

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/asutp"
)

// mockShutdownManager is a mock implementation of ShutdownManager
type mockShutdownManager struct {
	addShutdownFunc  func(ctx context.Context, req dto.AddShutdownRequest) (int64, error)
	editShutdownFunc func(ctx context.Context, id int64, req dto.EditShutdownRequest) error
	addCalls         []dto.AddShutdownRequest
	editCalls        []struct {
		ID  int64
		Req dto.EditShutdownRequest
	}
}

func (m *mockShutdownManager) AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error) {
	m.addCalls = append(m.addCalls, req)
	if m.addShutdownFunc != nil {
		return m.addShutdownFunc(ctx, req)
	}
	return int64(len(m.addCalls)), nil
}

func (m *mockShutdownManager) EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error {
	m.editCalls = append(m.editCalls, struct {
		ID  int64
		Req dto.EditShutdownRequest
	}{ID: id, Req: req})
	if m.editShutdownFunc != nil {
		return m.editShutdownFunc(ctx, id, req)
	}
	return nil
}

// mockStateTracker is a mock implementation of StateTracker
type mockStateTracker struct {
	activeShutdowns         map[string]int64
	getActiveShutdownFunc   func(ctx context.Context, stationID int64, deviceID string) (int64, error)
	setActiveShutdownFunc   func(ctx context.Context, stationID int64, deviceID string, shutdownID int64) error
	clearActiveShutdownFunc func(ctx context.Context, stationID int64, deviceID string) error
}

func newMockStateTracker() *mockStateTracker {
	return &mockStateTracker{
		activeShutdowns: make(map[string]int64),
	}
}

func (m *mockStateTracker) makeKey(stationID int64, deviceID string) string {
	return string(rune(stationID)) + ":" + deviceID
}

func (m *mockStateTracker) GetActiveShutdown(ctx context.Context, stationID int64, deviceID string) (int64, error) {
	if m.getActiveShutdownFunc != nil {
		return m.getActiveShutdownFunc(ctx, stationID, deviceID)
	}
	key := m.makeKey(stationID, deviceID)
	return m.activeShutdowns[key], nil
}

func (m *mockStateTracker) SetActiveShutdown(ctx context.Context, stationID int64, deviceID string, shutdownID int64) error {
	if m.setActiveShutdownFunc != nil {
		return m.setActiveShutdownFunc(ctx, stationID, deviceID, shutdownID)
	}
	key := m.makeKey(stationID, deviceID)
	m.activeShutdowns[key] = shutdownID
	return nil
}

func (m *mockStateTracker) ClearActiveShutdown(ctx context.Context, stationID int64, deviceID string) error {
	if m.clearActiveShutdownFunc != nil {
		return m.clearActiveShutdownFunc(ctx, stationID, deviceID)
	}
	key := m.makeKey(stationID, deviceID)
	delete(m.activeShutdowns, key)
	return nil
}

func TestProcessor_ProcessEnvelope_CreateShutdown(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	shutdownMgr := &mockShutdownManager{}
	stateTracker := newMockStateTracker()

	processor := NewProcessor(shutdownMgr, stateTracker, log)

	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	env := &asutp.Envelope{
		ID:          "test-1",
		StationID:   "ges1",
		StationName: "ГЭС-1",
		Timestamp:   timestamp,
		DeviceID:    "gen1",
		DeviceName:  "Генератор №1",
		DeviceGroup: "generators",
		Values: []asutp.DataPoint{
			{Name: "emergency_stop", Value: true, Quality: "good"},
			{Name: "protection_set_a_trip", Value: true, Quality: "good"},
		},
	}

	err := processor.ProcessEnvelope(ctx, 32, env)
	if err != nil {
		t.Fatalf("ProcessEnvelope() error = %v", err)
	}

	// Verify shutdown was created
	if len(shutdownMgr.addCalls) != 1 {
		t.Fatalf("Expected 1 AddShutdown call, got %d", len(shutdownMgr.addCalls))
	}

	req := shutdownMgr.addCalls[0]
	if req.OrganizationID != 32 {
		t.Errorf("OrganizationID = %d, want 32", req.OrganizationID)
	}
	if req.StartTime != timestamp {
		t.Errorf("StartTime = %v, want %v", req.StartTime, timestamp)
	}
	if req.EndTime != nil {
		t.Errorf("EndTime = %v, want nil", req.EndTime)
	}
	if req.Reason == nil {
		t.Fatal("Reason is nil")
	}
	expectedReason := "Г1: Аварийный останов, Срабатывание защиты комплекта А"
	if *req.Reason != expectedReason {
		t.Errorf("Reason = %q, want %q", *req.Reason, expectedReason)
	}
	if req.CreatedByUserID != SystemUserID {
		t.Errorf("CreatedByUserID = %d, want %d", req.CreatedByUserID, SystemUserID)
	}
}

func TestProcessor_ProcessEnvelope_NoDuplicateShutdown(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	shutdownMgr := &mockShutdownManager{}
	stateTracker := newMockStateTracker()

	// Simulate existing active shutdown
	stateTracker.activeShutdowns[stateTracker.makeKey(32, "gen1")] = 100

	processor := NewProcessor(shutdownMgr, stateTracker, log)

	env := &asutp.Envelope{
		ID:        "test-2",
		DeviceID:  "gen1",
		Timestamp: time.Now(),
		Values: []asutp.DataPoint{
			{Name: "emergency_stop", Value: true, Quality: "good"},
		},
	}

	err := processor.ProcessEnvelope(ctx, 32, env)
	if err != nil {
		t.Fatalf("ProcessEnvelope() error = %v", err)
	}

	// Verify no new shutdown was created (deduplication)
	if len(shutdownMgr.addCalls) != 0 {
		t.Errorf("Expected 0 AddShutdown calls, got %d", len(shutdownMgr.addCalls))
	}
}

func TestProcessor_ProcessEnvelope_CloseShutdown(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	shutdownMgr := &mockShutdownManager{}
	stateTracker := newMockStateTracker()

	// Simulate existing active shutdown
	stateTracker.activeShutdowns[stateTracker.makeKey(32, "gen1")] = 100

	processor := NewProcessor(shutdownMgr, stateTracker, log)

	endTime := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
	env := &asutp.Envelope{
		ID:        "test-3",
		DeviceID:  "gen1",
		Timestamp: endTime,
		Values: []asutp.DataPoint{
			{Name: "emergency_stop", Value: false, Quality: "good"},
			{Name: "temperature", Value: 50.0, Quality: "good"},
		},
	}

	err := processor.ProcessEnvelope(ctx, 32, env)
	if err != nil {
		t.Fatalf("ProcessEnvelope() error = %v", err)
	}

	// Verify shutdown was closed
	if len(shutdownMgr.editCalls) != 1 {
		t.Fatalf("Expected 1 EditShutdown call, got %d", len(shutdownMgr.editCalls))
	}

	editCall := shutdownMgr.editCalls[0]
	if editCall.ID != 100 {
		t.Errorf("EditShutdown ID = %d, want 100", editCall.ID)
	}
	if editCall.Req.EndTime == nil {
		t.Fatal("EndTime is nil")
	}
	if *editCall.Req.EndTime != endTime {
		t.Errorf("EndTime = %v, want %v", *editCall.Req.EndTime, endTime)
	}

	// Verify state was cleared
	key := stateTracker.makeKey(32, "gen1")
	if _, exists := stateTracker.activeShutdowns[key]; exists {
		t.Error("Active shutdown was not cleared from state tracker")
	}
}

func TestProcessor_ProcessEnvelope_NoAction(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	shutdownMgr := &mockShutdownManager{}
	stateTracker := newMockStateTracker()

	processor := NewProcessor(shutdownMgr, stateTracker, log)

	// No alarms, no active shutdown - should do nothing
	env := &asutp.Envelope{
		ID:        "test-4",
		DeviceID:  "gen1",
		Timestamp: time.Now(),
		Values: []asutp.DataPoint{
			{Name: "temperature", Value: 50.0, Quality: "good"},
			{Name: "pressure", Value: 100.0, Quality: "good"},
		},
	}

	err := processor.ProcessEnvelope(ctx, 32, env)
	if err != nil {
		t.Fatalf("ProcessEnvelope() error = %v", err)
	}

	if len(shutdownMgr.addCalls) != 0 {
		t.Errorf("Expected 0 AddShutdown calls, got %d", len(shutdownMgr.addCalls))
	}
	if len(shutdownMgr.editCalls) != 0 {
		t.Errorf("Expected 0 EditShutdown calls, got %d", len(shutdownMgr.editCalls))
	}
}

func TestProcessor_ProcessEnvelope_RedisErrorOnGet(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	shutdownMgr := &mockShutdownManager{}
	stateTracker := newMockStateTracker()
	stateTracker.getActiveShutdownFunc = func(ctx context.Context, stationID int64, deviceID string) (int64, error) {
		return 0, errors.New("redis connection error")
	}

	processor := NewProcessor(shutdownMgr, stateTracker, log)

	env := &asutp.Envelope{
		ID:        "test-5",
		DeviceID:  "gen1",
		Timestamp: time.Now(),
		Values: []asutp.DataPoint{
			{Name: "emergency_stop", Value: true, Quality: "good"},
		},
	}

	// Should still create shutdown despite Redis error
	err := processor.ProcessEnvelope(ctx, 32, env)
	if err != nil {
		t.Fatalf("ProcessEnvelope() error = %v", err)
	}

	if len(shutdownMgr.addCalls) != 1 {
		t.Errorf("Expected 1 AddShutdown call (even with Redis error), got %d", len(shutdownMgr.addCalls))
	}
}

func TestProcessor_ProcessEnvelope_ShutdownCreationError(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	shutdownMgr := &mockShutdownManager{
		addShutdownFunc: func(ctx context.Context, req dto.AddShutdownRequest) (int64, error) {
			return 0, errors.New("database error")
		},
	}
	stateTracker := newMockStateTracker()

	processor := NewProcessor(shutdownMgr, stateTracker, log)

	env := &asutp.Envelope{
		ID:        "test-6",
		DeviceID:  "gen1",
		Timestamp: time.Now(),
		Values: []asutp.DataPoint{
			{Name: "emergency_stop", Value: true, Quality: "good"},
		},
	}

	// Should not return error - just log and continue
	err := processor.ProcessEnvelope(ctx, 32, env)
	if err != nil {
		t.Fatalf("ProcessEnvelope() should not return error on shutdown creation failure, got %v", err)
	}
}
