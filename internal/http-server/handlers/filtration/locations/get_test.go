package locations

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/token"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
)

// mockTokenVerifier implements mwauth.TokenVerifier for injecting claims into context.
type mockTokenVerifier struct {
	claims *token.Claims
	err    error
}

func (m *mockTokenVerifier) Verify(_ string) (*token.Claims, error) {
	return m.claims, m.err
}

type mockLocationGetter struct {
	getFunc func(ctx context.Context, orgID int64) ([]filtration.Location, error)
}

func (m *mockLocationGetter) GetFiltrationLocationsByOrg(ctx context.Context, orgID int64) ([]filtration.Location, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, orgID)
	}
	return nil, nil
}

// withAuth wraps a handler with the Authenticator middleware and a mock verifier
// that returns the given claims. The request must include "Authorization: Bearer test-token".
func withAuth(handler http.HandlerFunc, claims *token.Claims) http.Handler {
	verifier := &mockTokenVerifier{claims: claims}
	return mwauth.Authenticator(verifier)(handler)
}

func TestGet(t *testing.T) {
	// Claims for a user with "sc" role (full access)
	scClaims := &token.Claims{
		UserID:         1,
		OrganizationID: 0,
		Roles:          []string{"sc"},
	}

	tests := []struct {
		name           string
		url            string
		claims         *token.Claims
		mockResult     []filtration.Location
		mockError      error
		wantStatusCode int
	}{
		{
			name:           "missing organization_id",
			url:            "/filtration/locations",
			claims:         scClaims,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid organization_id",
			url:            "/filtration/locations?organization_id=abc",
			claims:         scClaims,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "successful get",
			url:            "/filtration/locations?organization_id=1",
			claims:         scClaims,
			mockResult:     []filtration.Location{{ID: 1, Name: "Location 1"}},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "internal server error",
			url:            "/filtration/locations?organization_id=1",
			claims:         scClaims,
			mockError:      errors.New("database error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLocationGetter{
				getFunc: func(ctx context.Context, orgID int64) ([]filtration.Location, error) {
					return tt.mockResult, tt.mockError
				},
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := withAuth(Get(logger, mock), tt.claims)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestGetOrgAccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	successLocations := []filtration.Location{
		{ID: 1, Name: "Location 1", OrganizationID: 5},
	}
	mock := &mockLocationGetter{
		getFunc: func(ctx context.Context, orgID int64) ([]filtration.Location, error) {
			return successLocations, nil
		},
	}

	tests := []struct {
		name           string
		url            string
		claims         *token.Claims
		wantStatusCode int
	}{
		{
			name: "sc role — access any org",
			url:  "/filtration/locations?organization_id=5",
			claims: &token.Claims{
				UserID:         1,
				OrganizationID: 0,
				Roles:          []string{"sc"},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "reservoir role — own org allowed",
			url:  "/filtration/locations?organization_id=5",
			claims: &token.Claims{
				UserID:         2,
				OrganizationID: 5,
				Roles:          []string{"reservoir"},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "reservoir role — different org forbidden",
			url:  "/filtration/locations?organization_id=5",
			claims: &token.Claims{
				UserID:         3,
				OrganizationID: 10,
				Roles:          []string{"reservoir"},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "reservoir role — no org assigned forbidden",
			url:  "/filtration/locations?organization_id=5",
			claims: &token.Claims{
				UserID:         4,
				OrganizationID: 0,
				Roles:          []string{"reservoir"},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "rais role — access any org",
			url:  "/filtration/locations?organization_id=5",
			claims: &token.Claims{
				UserID:         5,
				OrganizationID: 0,
				Roles:          []string{"rais"},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			verifier := &mockTokenVerifier{claims: tt.claims}
			r.Use(mwauth.Authenticator(verifier))
			r.Get("/filtration/locations", Get(logger, mock))

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("%s: got status %d, want %d", tt.name, rr.Code, tt.wantStatusCode)
			}
		})
	}
}
