package add

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"testing"
	"time"
)

type mockDischargeAdder struct {
	addFunc  func(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error)
	linkFunc func(ctx context.Context, dischargeID int64, fileIDs []int64) error
	addCalls int
}

func (m *mockDischargeAdder) AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error) {
	m.addCalls++
	if m.addFunc != nil {
		return m.addFunc(ctx, orgID, createdByID, startTime, endTime, flowRate, reason)
	}
	return 1, nil
}

func (m *mockDischargeAdder) LinkDischargeFiles(ctx context.Context, dischargeID int64, fileIDs []int64) error {
	if m.linkFunc != nil {
		return m.linkFunc(ctx, dischargeID, fileIDs)
	}
	return nil
}

type mockOngoingChecker struct {
	ensureFunc func(ctx context.Context, orgID int64, force bool, newStartTime time.Time) error
}

func (m *mockOngoingChecker) EnsureNoOngoingDischarge(ctx context.Context, orgID int64, force bool, newStartTime time.Time) error {
	if m.ensureFunc != nil {
		return m.ensureFunc(ctx, orgID, force, newStartTime)
	}
	return nil
}

// mockBackdateRotator records calls to RotateBackdatedDischarge. Default
// behavior: passthrough (returns dischargeID unchanged) — matches the
// "no rotation needed" path so existing tests don't need to set rotateFunc.
type mockBackdateRotator struct {
	rotateFunc func(ctx context.Context, dischargeID int64, cutoffs []time.Time) (int64, error)
	gotID      int64
	gotCutoffs []time.Time
	calls      int
}

func (m *mockBackdateRotator) RotateBackdatedDischarge(ctx context.Context, dischargeID int64, cutoffs []time.Time) (int64, error) {
	m.calls++
	m.gotID = dischargeID
	m.gotCutoffs = cutoffs
	if m.rotateFunc != nil {
		return m.rotateFunc(ctx, dischargeID, cutoffs)
	}
	return dischargeID, nil
}

func contextWithClaims(ctx context.Context, userID int64) context.Context {
	claims := &token.Claims{
		UserID: userID,
		Name:   "Test User",
		Roles:  []string{"sc"},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

func TestNew_OngoingDischargeConflict(t *testing.T) {
	now := time.Now()

	checker := &mockOngoingChecker{
		ensureFunc: func(_ context.Context, _ int64, force bool, _ time.Time) error {
			if !force {
				return storage.ErrOngoingDischargeExists
			}
			return nil
		},
	}
	adder := &mockDischargeAdder{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("conflict without force", func(t *testing.T) {
		body := Request{
			OrganizationID: 1,
			StartedAt:      now,
			FlowRate:       5.0,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/discharges", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithClaims(req.Context(), 1))

		rr := httptest.NewRecorder()
		handler := New(logger, adder, checker, &mockBackdateRotator{}, time.UTC)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Errorf("expected status %d, got %d", http.StatusConflict, rr.Code)
		}
	})

	t.Run("force closes existing and creates new", func(t *testing.T) {
		body := Request{
			OrganizationID: 1,
			StartedAt:      now,
			FlowRate:       5.0,
			Force:          true,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/discharges", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithClaims(req.Context(), 1))

		rr := httptest.NewRecorder()
		handler := New(logger, adder, checker, &mockBackdateRotator{}, time.UTC)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}
	})

	t.Run("no ongoing discharge proceeds normally", func(t *testing.T) {
		noConflictChecker := &mockOngoingChecker{
			ensureFunc: func(_ context.Context, _ int64, _ bool, _ time.Time) error {
				return nil
			},
		}

		body := Request{
			OrganizationID: 1,
			StartedAt:      now,
			FlowRate:       5.0,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/discharges", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithClaims(req.Context(), 1))

		rr := httptest.NewRecorder()
		handler := New(logger, adder, noConflictChecker, &mockBackdateRotator{}, time.UTC)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}
	})
}

// TestNew_BackdateRotation covers the cutoff-aware response.ID behavior.
// loc=Asia/Tashkent so the cutoffs are at 05:00 local. Since happy-path
// existing tests use `now` for StartedAt, those continue to skip rotation.
func TestNew_BackdateRotation(t *testing.T) {
	tashkent := time.FixedZone("Asia/Tashkent", 5*3600)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	adder := &mockDischargeAdder{
		addFunc: func(_ context.Context, _, _ int64, _ time.Time, _ *time.Time, _ float64, _ *string) (int64, error) {
			return 100, nil // original ID
		},
	}
	noConflictChecker := &mockOngoingChecker{}

	post := func(t *testing.T, started time.Time, rotator *mockBackdateRotator) *httptest.ResponseRecorder {
		t.Helper()
		body := Request{OrganizationID: 1, StartedAt: started, FlowRate: 5.0}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/discharges", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithClaims(req.Context(), 1))
		rr := httptest.NewRecorder()
		handler := New(logger, adder, noConflictChecker, rotator, tashkent)
		handler.ServeHTTP(rr, req)
		return rr
	}

	decodeID := func(t *testing.T, body string) int64 {
		t.Helper()
		var resp struct{ ID int64 `json:"id"` }
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return resp.ID
	}

	t.Run("no backdate: rotator NOT called, response.ID = original", func(t *testing.T) {
		// StartedAt = now → ComputeCutoffs returns 0 cutoffs.
		now := time.Now().In(tashkent)
		rotator := &mockBackdateRotator{}
		rr := post(t, now, rotator)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status: want 201, got %d body=%s", rr.Code, rr.Body.String())
		}
		if rotator.calls != 0 {
			t.Errorf("rotator should NOT be called; got %d calls", rotator.calls)
		}
		if got := decodeID(t, rr.Body.String()); got != 100 {
			t.Errorf("response.ID: want 100 (original), got %d", got)
		}
	})

	t.Run("backdate to before today's 05:00: rotator called, response.ID = final clone", func(t *testing.T) {
		// Pick a started_at clearly before today's 05:00 Tashkent. Compute it
		// as "yesterday at 02:00 in tashkent" to stay deterministic regardless
		// of when the test runs.
		now := time.Now().In(tashkent)
		yest := time.Date(now.Year(), now.Month(), now.Day()-1, 2, 0, 0, 0, tashkent)
		rotator := &mockBackdateRotator{
			rotateFunc: func(_ context.Context, dischargeID int64, cutoffs []time.Time) (int64, error) {
				if dischargeID != 100 {
					t.Errorf("rotator got dischargeID %d, want 100", dischargeID)
				}
				if len(cutoffs) < 1 {
					t.Errorf("expected >=1 cutoff for yesterday-02:00 backdate, got %d", len(cutoffs))
				}
				return 200, nil // final clone
			},
		}
		rr := post(t, yest, rotator)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status: want 201, got %d body=%s", rr.Code, rr.Body.String())
		}
		if rotator.calls != 1 {
			t.Errorf("rotator should be called once; got %d", rotator.calls)
		}
		if got := decodeID(t, rr.Body.String()); got != 200 {
			t.Errorf("response.ID: want 200 (final clone), got %d", got)
		}
	})

	t.Run("backdate too old: 400 ErrBackdateTooOld; AddDischarge NOT called either", func(t *testing.T) {
		// Use a fresh adder so addCalls doesn't include the prior subtests'
		// calls. Pinning AddDischarge NOT called is important — it ensures
		// the too-old check fires BEFORE the DB write (no orphan rows).
		now := time.Now().In(tashkent)
		tooOld := time.Date(now.Year(), now.Month(), now.Day()-200, 2, 0, 0, 0, tashkent)
		freshAdder := &mockDischargeAdder{
			addFunc: func(_ context.Context, _, _ int64, _ time.Time, _ *time.Time, _ float64, _ *string) (int64, error) {
				return 100, nil
			},
		}
		rotator := &mockBackdateRotator{}
		body := Request{OrganizationID: 1, StartedAt: tooOld, FlowRate: 5.0}
		bodyBytes, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/discharges", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(contextWithClaims(req.Context(), 1))
		rr := httptest.NewRecorder()
		handler := New(logger, freshAdder, noConflictChecker, rotator, tashkent)
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status: want 400, got %d body=%s", rr.Code, rr.Body.String())
		}
		if rotator.calls != 0 {
			t.Errorf("rotator should NOT be called for too-old start; got %d", rotator.calls)
		}
		if freshAdder.addCalls != 0 {
			t.Errorf("AddDischarge should NOT be called for too-old start; got %d calls", freshAdder.addCalls)
		}
	})
}
