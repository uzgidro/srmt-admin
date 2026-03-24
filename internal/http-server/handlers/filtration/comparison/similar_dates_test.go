package comparison

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

type mockTokenVerifier struct {
	claims *token.Claims
	err    error
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, m.err
}

func withAuth(handler http.HandlerFunc, claims *token.Claims) http.Handler {
	verifier := &mockTokenVerifier{claims: claims}
	return mwauth.Authenticator(verifier)(handler)
}

type mockSimilarDatesGetter struct {
	orgIDs          []int64
	orgIDsErr       error
	orgName         string
	orgNameErr      error
	level           *float64
	volume          *float64
	levelVolumeErr  error
	similarDates    []filtration.SimilarDate
	similarDatesErr error
}

func (m *mockSimilarDatesGetter) GetFiltrationOrgIDs(_ context.Context) ([]int64, error) {
	return m.orgIDs, m.orgIDsErr
}

func (m *mockSimilarDatesGetter) GetReservoirLevelVolume(_ context.Context, _ int64, _ string) (*float64, *float64, error) {
	return m.level, m.volume, m.levelVolumeErr
}

func (m *mockSimilarDatesGetter) GetSimilarLevelDates(_ context.Context, _ int64, _ float64, _ string, _ int) ([]filtration.SimilarDate, error) {
	return m.similarDates, m.similarDatesErr
}

func (m *mockSimilarDatesGetter) GetOrganizationName(_ context.Context, _ int64) (string, error) {
	return m.orgName, m.orgNameErr
}

func TestGetSimilarDates(t *testing.T) {
	scClaims := &token.Claims{UserID: 1, Roles: []string{"sc"}}
	reservoirClaims := &token.Claims{UserID: 2, Roles: []string{"reservoir"}, OrganizationID: 5}
	noOrgClaims := &token.Claims{UserID: 3, Roles: []string{"reservoir"}}

	lvl := 284.5
	vol := 12.3

	tests := []struct {
		name       string
		url        string
		claims     *token.Claims
		mock       *mockSimilarDatesGetter
		wantStatus int
		wantErr    bool
		wantLen    int // expected number of orgs in result
	}{
		{
			name:       "missing date",
			url:        "/filtration/comparison/similar-dates",
			claims:     scClaims,
			mock:       &mockSimilarDatesGetter{},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid date format",
			url:        "/filtration/comparison/similar-dates?date=bad",
			claims:     scClaims,
			mock:       &mockSimilarDatesGetter{},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "no organization assigned",
			url:        "/filtration/comparison/similar-dates?date=2025-01-01",
			claims:     noOrgClaims,
			mock:       &mockSimilarDatesGetter{},
			wantStatus: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:   "supervisor — org IDs error",
			url:    "/filtration/comparison/similar-dates?date=2025-01-01",
			claims: scClaims,
			mock: &mockSimilarDatesGetter{
				orgIDsErr: errors.New("db error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:   "supervisor — no reservoir data (level nil)",
			url:    "/filtration/comparison/similar-dates?date=2025-01-01",
			claims: scClaims,
			mock: &mockSimilarDatesGetter{
				orgIDs:  []int64{1},
				orgName: "Org A",
				level:   nil,
				volume:  nil,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:   "supervisor — success with similar dates",
			url:    "/filtration/comparison/similar-dates?date=2025-01-01&limit=5",
			claims: scClaims,
			mock: &mockSimilarDatesGetter{
				orgIDs:  []int64{1},
				orgName: "Org A",
				level:   &lvl,
				volume:  &vol,
				similarDates: []filtration.SimilarDate{
					{Date: "2024-11-15", Level: &lvl, Volume: &vol, LevelDelta: 0.02},
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:   "regular user — success",
			url:    "/filtration/comparison/similar-dates?date=2025-01-01",
			claims: reservoirClaims,
			mock: &mockSimilarDatesGetter{
				orgName:      "My Org",
				level:        &lvl,
				volume:       &vol,
				similarDates: []filtration.SimilarDate{},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := withAuth(GetSimilarDates(log, tt.mock), tt.claims)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantErr {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("expected error in body, got: %v", resp)
				}
				return
			}

			var result []filtration.OrgSimilarDates
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("result len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}
