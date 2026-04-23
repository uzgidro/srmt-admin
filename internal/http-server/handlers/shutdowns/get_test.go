package shutdowns

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/token"
	"testing"
	"time"
)

// mockShutdownListGetter implements the shutdownGetter interface plus the
// upcoming GetShutdownsByCascade method. Counters expose call activity so
// tests can guard against accidental branch swaps in production code.
type mockShutdownListGetter struct {
	getAllFunc       func(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
	getCascadeFunc   func(ctx context.Context, day time.Time, cascadeOrgID int64) ([]*shutdown.ResponseModel, error)
	typesFunc        func(ctx context.Context) (map[int64][]string, error)
	getAllCalls      int
	getCascadeCalls  int
	typesCalls       int
	lastCascadeOrgID int64
}

func (m *mockShutdownListGetter) GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
	m.getAllCalls++
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx, day)
	}
	return []*shutdown.ResponseModel{}, nil
}

func (m *mockShutdownListGetter) GetShutdownsByCascade(ctx context.Context, day time.Time, cascadeOrgID int64) ([]*shutdown.ResponseModel, error) {
	m.getCascadeCalls++
	m.lastCascadeOrgID = cascadeOrgID
	if m.getCascadeFunc != nil {
		return m.getCascadeFunc(ctx, day, cascadeOrgID)
	}
	return []*shutdown.ResponseModel{}, nil
}

func (m *mockShutdownListGetter) GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error) {
	m.typesCalls++
	if m.typesFunc != nil {
		return m.typesFunc(ctx)
	}
	return map[int64][]string{}, nil
}

// nopMinio is a stub MinioURLGenerator that returns a fixed empty URL without
// touching the network. Sufficient because tests focus on routing/auth, not
// file presigning.
type nopMinio struct{}

func (n *nopMinio) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (*url.URL, error) {
	return &url.URL{}, nil
}

func runGet(t *testing.T, mock *mockShutdownListGetter, claims *token.Claims, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/shutdowns?"+query, nil)
	if claims != nil {
		req = req.WithContext(mwauth.ContextWithClaims(req.Context(), claims))
	}
	rr := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	minio := &nopMinio{}
	handler := Get(logger, mock, minio, time.UTC)
	handler.ServeHTTP(rr, req)
	return rr
}

func TestGet_NoClaims_FallbackToAll(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, nil, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getAllCalls != 1 {
		t.Errorf("GetShutdowns calls = %d, want 1", mock.getAllCalls)
	}
	if mock.getCascadeCalls != 0 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 0", mock.getCascadeCalls)
	}
}

func TestGet_ScUser_SeesAll(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 99, Roles: []string{"sc"}}, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getAllCalls != 1 {
		t.Errorf("GetShutdowns calls = %d, want 1 (sc must see all)", mock.getAllCalls)
	}
	if mock.getCascadeCalls != 0 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 0 (sc must NOT use cascade filter)", mock.getCascadeCalls)
	}
}

func TestGet_RaisUser_SeesAll(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 99, Roles: []string{"rais"}}, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getAllCalls != 1 {
		t.Errorf("GetShutdowns calls = %d, want 1 (rais must see all)", mock.getAllCalls)
	}
	if mock.getCascadeCalls != 0 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 0 (rais must NOT use cascade filter)", mock.getCascadeCalls)
	}
}

func TestGet_NonCascadeNonSc_SeesAll(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 99, Roles: []string{"reservoir"}}, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getAllCalls != 1 {
		t.Errorf("GetShutdowns calls = %d, want 1 (non-cascade roles fall back to all)", mock.getAllCalls)
	}
	if mock.getCascadeCalls != 0 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 0", mock.getCascadeCalls)
	}
}

func TestGet_CascadeUser_SeesOwnCascadeOnly(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 5, Roles: []string{"cascade"}}, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getCascadeCalls != 1 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 1", mock.getCascadeCalls)
	}
	if mock.lastCascadeOrgID != 5 {
		t.Errorf("cascade orgID = %d, want 5", mock.lastCascadeOrgID)
	}
	if mock.getAllCalls != 0 {
		t.Errorf("GetShutdowns calls = %d, want 0 (cascade user must NOT see all)", mock.getAllCalls)
	}
}

// Cascade user without an OrganizationID would otherwise leak the entire list
// because GetShutdownsByCascade(ctx, day, 0) would match no cascade. Handler
// must short-circuit and return an empty grouped response.
// Note: typesCalls intentionally not asserted — handler may or may not fetch
// the org-types map for an empty result; both are harmless.
func TestGet_CascadeUser_NoOrgID_EmptyList(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 0, Roles: []string{"cascade"}}, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getAllCalls != 0 {
		t.Errorf("GetShutdowns calls = %d, want 0 (cascade with no org must short-circuit)", mock.getAllCalls)
	}
	if mock.getCascadeCalls != 0 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 0 (cascade with no org must short-circuit)", mock.getCascadeCalls)
	}

	var resp shutdown.GroupedResponseWithURLs
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v. body=%s", err, rr.Body.String())
	}
	if len(resp.Ges) != 0 {
		t.Errorf("ges = %d items, want 0", len(resp.Ges))
	}
	if len(resp.Mini) != 0 {
		t.Errorf("mini = %d items, want 0", len(resp.Mini))
	}
	if len(resp.Micro) != 0 {
		t.Errorf("micro = %d items, want 0", len(resp.Micro))
	}
	if len(resp.Other) != 0 {
		t.Errorf("other = %d items, want 0", len(resp.Other))
	}
}

func TestGet_CascadeWithRaisRole_SeesAll(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 5, Roles: []string{"cascade", "rais"}}, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getAllCalls != 1 {
		t.Errorf("GetShutdowns calls = %d, want 1 (rais role overrides cascade filter)", mock.getAllCalls)
	}
	if mock.getCascadeCalls != 0 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 0 (rais role overrides cascade filter)", mock.getCascadeCalls)
	}
}

func TestGet_CascadeWithReservoirRole_SeesOwnOnly(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 5, Roles: []string{"cascade", "reservoir"}}, "date=2026-04-23")

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getCascadeCalls != 1 {
		t.Errorf("GetShutdownsByCascade calls = %d, want 1 (cascade wins over fallback)", mock.getCascadeCalls)
	}
	if mock.lastCascadeOrgID != 5 {
		t.Errorf("cascade orgID = %d, want 5", mock.lastCascadeOrgID)
	}
	if mock.getAllCalls != 0 {
		t.Errorf("GetShutdowns calls = %d, want 0 (cascade wins over fallback)", mock.getAllCalls)
	}
}

func TestGet_InvalidDateFormat(t *testing.T) {
	mock := &mockShutdownListGetter{}
	rr := runGet(t, mock, &token.Claims{UserID: 1, OrganizationID: 99, Roles: []string{"sc"}}, "date=not-a-date")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400. body=%s", rr.Code, rr.Body.String())
	}
	if mock.getAllCalls != 0 {
		t.Errorf("GetShutdowns must not be called on invalid date, got %d calls", mock.getAllCalls)
	}
	if mock.getCascadeCalls != 0 {
		t.Errorf("GetShutdownsByCascade must not be called on invalid date, got %d calls", mock.getCascadeCalls)
	}
}
