package reservoirdevicesummary

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/token"
)

type mockTokenVerifier struct {
	claims *token.Claims
	err    error
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, m.err
}

type mockPatcher struct{}

func (m *mockPatcher) PatchReservoirDeviceSummary(_ context.Context, _ dto.PatchReservoirDeviceSummaryRequest, _ int64) error {
	return nil
}

func TestPatchOrgAccess(t *testing.T) {
	tests := []struct {
		name       string
		claims     *token.Claims
		body       string
		wantStatus int
	}{
		{
			name: "sc role - access to any org",
			claims: &token.Claims{
				UserID:         1,
				OrganizationID: 1,
				Roles:          []string{"sc"},
			},
			body:       `{"updates": [{"organization_id": 999, "count_total": 10}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name: "reservoir role - own org",
			claims: &token.Claims{
				UserID:         2,
				OrganizationID: 5,
				Roles:          []string{"reservoir"},
			},
			body:       `{"updates": [{"organization_id": 5, "count_total": 10}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name: "reservoir role - foreign org",
			claims: &token.Claims{
				UserID:         3,
				OrganizationID: 5,
				Roles:          []string{"reservoir"},
			},
			body:       `{"updates": [{"organization_id": 10, "count_total": 10}]}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := slog.New(slog.NewTextHandler(os.Stdout, nil))
			handler := Patch(log, &mockPatcher{})

			verifier := &mockTokenVerifier{claims: tt.claims}

			r := chi.NewRouter()
			r.Use(mwauth.Authenticator(verifier))
			r.Patch("/reservoir-device", handler)

			req := httptest.NewRequest(http.MethodPatch, "/reservoir-device", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.wantStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
