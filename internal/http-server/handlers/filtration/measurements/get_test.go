package measurements

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/token"
)

// mockTokenVerifier implements mwauth.TokenVerifier for injecting claims into context.
type mockTokenVerifier struct {
	claims *token.Claims
	err    error
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, m.err
}

// withAuth wraps a handler with the Authenticator middleware and a mock verifier
// that returns the given claims. The request must include "Authorization: Bearer test-token".
func withAuth(handler http.HandlerFunc, claims *token.Claims) http.Handler {
	verifier := &mockTokenVerifier{claims: claims}
	return mwauth.Authenticator(verifier)(handler)
}

type mockFiltrationGetter struct {
	getFunc func(ctx context.Context, orgID int64, date string) ([]filtration.FiltrationMeasurement, error)
}

func (m *mockFiltrationGetter) GetFiltrationMeasurements(ctx context.Context, orgID int64, date string) ([]filtration.FiltrationMeasurement, error) {
	return m.getFunc(ctx, orgID, date)
}

type mockPiezometerGetter struct {
	getFunc func(ctx context.Context, orgID int64, date string) ([]filtration.PiezometerMeasurement, error)
}

func (m *mockPiezometerGetter) GetPiezometerMeasurements(ctx context.Context, orgID int64, date string) ([]filtration.PiezometerMeasurement, error) {
	return m.getFunc(ctx, orgID, date)
}

func TestGet(t *testing.T) {
	flowRate := 1.5
	level := 2.0

	scClaims := &token.Claims{
		UserID: 1,
		Roles:  []string{"sc"},
	}

	tests := []struct {
		name             string
		url              string
		filtrationReturn []filtration.FiltrationMeasurement
		filtrationErr    error
		piezometerReturn []filtration.PiezometerMeasurement
		piezometerErr    error
		wantStatusCode   int
		wantErrInBody    bool
	}{
		{
			name:           "missing organization_id",
			url:            "/filtration/measurements?date=2025-01-01",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name:           "invalid organization_id",
			url:            "/filtration/measurements?organization_id=abc&date=2025-01-01",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name:           "missing date",
			url:            "/filtration/measurements?organization_id=1",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
		},
		{
			name: "successful get",
			url:  "/filtration/measurements?organization_id=1&date=2025-01-01",
			filtrationReturn: []filtration.FiltrationMeasurement{
				{ID: 1, LocationID: 10, Date: "2025-01-01", FlowRate: &flowRate},
			},
			piezometerReturn: []filtration.PiezometerMeasurement{
				{ID: 2, PiezometerID: 20, Date: "2025-01-01", Level: &level},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "filtration getter error",
			url:            "/filtration/measurements?organization_id=1&date=2025-01-01",
			filtrationErr:  errors.New("db error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
		},
		{
			name: "piezometer getter error",
			url:  "/filtration/measurements?organization_id=1&date=2025-01-01",
			filtrationReturn: []filtration.FiltrationMeasurement{
				{ID: 1, LocationID: 10, Date: "2025-01-01", FlowRate: &flowRate},
			},
			piezometerErr:  errors.New("db error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fg := &mockFiltrationGetter{
				getFunc: func(_ context.Context, _ int64, _ string) ([]filtration.FiltrationMeasurement, error) {
					return tt.filtrationReturn, tt.filtrationErr
				},
			}
			pg := &mockPiezometerGetter{
				getFunc: func(_ context.Context, _ int64, _ string) ([]filtration.PiezometerMeasurement, error) {
					return tt.piezometerReturn, tt.piezometerErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := withAuth(Get(log, fg, pg), scClaims)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatusCode)
			}

			if tt.wantErrInBody {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("expected error in body, got: %v", resp)
				}
				return
			}

			var result GetResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(result.FiltrationMeasurements) != len(tt.filtrationReturn) {
				t.Errorf("filtration measurements len = %d, want %d", len(result.FiltrationMeasurements), len(tt.filtrationReturn))
			}
			if len(result.PiezometerMeasurements) != len(tt.piezometerReturn) {
				t.Errorf("piezometer measurements len = %d, want %d", len(result.PiezometerMeasurements), len(tt.piezometerReturn))
			}
		})
	}
}
