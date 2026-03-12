package comparison

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

type mockTokenVerifier struct {
	claims *token.Claims
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, nil
}

func withAuth(handler http.HandlerFunc, claims *token.Claims) http.Handler {
	verifier := &mockTokenVerifier{claims: claims}
	return mwauth.Authenticator(verifier)(handler)
}

type mockGetter struct {
	orgIDs        []int64
	orgIDsErr     error
	summaries     map[string]*filtration.OrgFiltrationSummary // key: "orgID:date"
	summaryErr    error
	levels        map[string][2]*float64 // key: "orgID:date" → [level, volume]
	levelErr      error
	closestDate   string
	closestDateErr error
}

func key(orgID int64, date string) string {
	return fmt.Sprintf("%d:%s", orgID, date)
}

func (m *mockGetter) GetFiltrationOrgIDs(_ context.Context) ([]int64, error) {
	return m.orgIDs, m.orgIDsErr
}

func (m *mockGetter) GetOrgFiltrationSummary(_ context.Context, orgID int64, date string) (*filtration.OrgFiltrationSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	s, ok := m.summaries[key(orgID, date)]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return s, nil
}

func (m *mockGetter) GetReservoirLevelVolume(_ context.Context, orgID int64, date string) (*float64, *float64, error) {
	if m.levelErr != nil {
		return nil, nil, m.levelErr
	}
	lv, ok := m.levels[key(orgID, date)]
	if !ok {
		return nil, nil, nil
	}
	return lv[0], lv[1], nil
}

func (m *mockGetter) GetClosestLevelDate(_ context.Context, _ int64, _ float64, _ string) (string, error) {
	return m.closestDate, m.closestDateErr
}

func ptr(v float64) *float64 { return &v }

var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestGet_MissingDate(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/filtration/comparison", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	claims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	handler := withAuth(Get(discardLog, &mockGetter{}), claims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestGet_InvalidDateFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/filtration/comparison?date=foobar", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	claims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	handler := withAuth(Get(discardLog, &mockGetter{}), claims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestGet_ReservoirRoleNoOrg(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/filtration/comparison?date=2025-01-01", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	claims := &token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationID: 0}
	handler := withAuth(Get(discardLog, &mockGetter{}), claims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestGet_SupervisorSuccess(t *testing.T) {
	level := 100.5
	volume := 50.0
	histLevel := 100.3
	histVolume := 49.8

	mock := &mockGetter{
		orgIDs: []int64{1},
		summaries: map[string]*filtration.OrgFiltrationSummary{
			"1:2025-01-01": {
				OrganizationID:   1,
				OrganizationName: "Test Reservoir",
				Locations:        []filtration.LocationReading{},
				Piezometers:      []filtration.PiezoReading{},
				PiezoCounts:      filtration.PiezometerCounts{Pressure: 2},
			},
			"1:2024-06-15": {
				OrganizationID:   1,
				OrganizationName: "Test Reservoir",
				Locations:        []filtration.LocationReading{},
				Piezometers:      []filtration.PiezoReading{},
				PiezoCounts:      filtration.PiezometerCounts{Pressure: 2},
			},
		},
		levels: map[string][2]*float64{
			"1:2025-01-01": {&level, &volume},
			"1:2024-06-15": {&histLevel, &histVolume},
		},
		closestDate: "2024-06-15",
	}

	req := httptest.NewRequest(http.MethodGet, "/filtration/comparison?date=2025-01-01", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	claims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	handler := withAuth(Get(discardLog, mock), claims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var result []filtration.OrgComparison
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("result length = %d, want 1", len(result))
	}
	if result[0].OrganizationID != 1 {
		t.Errorf("org_id = %d, want 1", result[0].OrganizationID)
	}
	if result[0].Current.Date != "2025-01-01" {
		t.Errorf("current date = %q, want 2025-01-01", result[0].Current.Date)
	}
	if result[0].Historical == nil {
		t.Fatal("expected historical snapshot, got nil")
	}
	if result[0].Historical.Date != "2024-06-15" {
		t.Errorf("historical date = %q, want 2024-06-15", result[0].Historical.Date)
	}
}

func TestGet_ReservoirRoleOwnOrg(t *testing.T) {
	mock := &mockGetter{
		summaries: map[string]*filtration.OrgFiltrationSummary{
			"5:2025-03-01": {
				OrganizationID:   5,
				OrganizationName: "My Reservoir",
				Locations:        []filtration.LocationReading{},
				Piezometers:      []filtration.PiezoReading{},
			},
		},
		levels:         map[string][2]*float64{},
		closestDateErr: storage.ErrNotFound,
	}

	req := httptest.NewRequest(http.MethodGet, "/filtration/comparison?date=2025-03-01", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	claims := &token.Claims{UserID: 1, Roles: []string{"reservoir"}, OrganizationID: 5}
	handler := withAuth(Get(discardLog, mock), claims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var result []filtration.OrgComparison
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("result length = %d, want 1", len(result))
	}
	if result[0].OrganizationID != 5 {
		t.Errorf("org_id = %d, want 5", result[0].OrganizationID)
	}
	if result[0].Historical != nil {
		t.Error("expected no historical snapshot when level is nil")
	}
}

func TestGet_NoHistoricalData(t *testing.T) {
	level := 100.0
	mock := &mockGetter{
		orgIDs: []int64{1},
		summaries: map[string]*filtration.OrgFiltrationSummary{
			"1:2025-01-01": {
				OrganizationID:   1,
				OrganizationName: "Test",
				Locations:        []filtration.LocationReading{},
				Piezometers:      []filtration.PiezoReading{},
			},
		},
		levels: map[string][2]*float64{
			"1:2025-01-01": {&level, nil},
		},
		closestDateErr: storage.ErrNotFound,
	}

	req := httptest.NewRequest(http.MethodGet, "/filtration/comparison?date=2025-01-01", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	claims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	handler := withAuth(Get(discardLog, mock), claims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var result []filtration.OrgComparison
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("result length = %d, want 1", len(result))
	}
	if result[0].Historical != nil {
		t.Error("expected nil historical when no closest level found")
	}
}

func TestGet_OrgIDsError(t *testing.T) {
	mock := &mockGetter{
		orgIDsErr: errors.New("db error"),
	}

	req := httptest.NewRequest(http.MethodGet, "/filtration/comparison?date=2025-01-01", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()

	claims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	handler := withAuth(Get(discardLog, mock), claims)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
