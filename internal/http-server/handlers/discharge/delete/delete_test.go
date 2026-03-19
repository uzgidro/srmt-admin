package delete

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/token"
)

type mockTokenVerifier struct {
	claims *token.Claims
	err    error
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, m.err
}

type mockDischargeDeleter struct {
	err error
}

func (m *mockDischargeDeleter) DeleteDischarge(_ context.Context, _ int64) error {
	return m.err
}

type mockDischargeOrgGetter struct {
	orgID int64
	err   error
}

func (m *mockDischargeOrgGetter) GetDischargeOrgID(_ context.Context, _ int64) (int64, error) {
	return m.orgID, m.err
}

func TestDeleteOrgAccess(t *testing.T) {
	tests := []struct {
		name           string
		claims         *token.Claims
		resourceOrganizationID  int64
		expectedStatus int
	}{
		{
			name: "sc role - access to any org",
			claims: &token.Claims{
				Roles: []string{"sc"},
				OrganizationID: 1,
			},
			resourceOrganizationID:  999,
			expectedStatus: http.StatusOK,
		},
		{
			name: "reservoir role - own org",
			claims: &token.Claims{
				Roles: []string{"reservoir"},
				OrganizationID: 5,
			},
			resourceOrganizationID:  5,
			expectedStatus: http.StatusOK,
		},
		{
			name: "reservoir role - foreign org",
			claims: &token.Claims{
				Roles: []string{"reservoir"},
				OrganizationID: 5,
			},
			resourceOrganizationID:  10,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			verifier := &mockTokenVerifier{claims: tt.claims}
			deleter := &mockDischargeDeleter{err: nil}
			orgGetter := &mockDischargeOrgGetter{orgID: tt.resourceOrganizationID}

			r := chi.NewRouter()
			r.Use(mwauth.Authenticator(verifier))
			r.Delete("/discharges/{id}", New(logger, deleter, orgGetter))

			req := httptest.NewRequest(http.MethodDelete, "/discharges/1", nil)
			req.Header.Set("Authorization", "Bearer test-token")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
