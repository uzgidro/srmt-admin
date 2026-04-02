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
}

func (m *mockDischargeAdder) AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error) {
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
		handler := New(logger, adder, checker, nil, nil, nil)
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
		handler := New(logger, adder, checker, nil, nil, nil)
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
		handler := New(logger, adder, noConflictChecker, nil, nil, nil)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}
	})
}
