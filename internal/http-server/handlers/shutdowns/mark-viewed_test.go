package shutdowns

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"testing"

	"github.com/go-chi/chi/v5"
)

type mockShutdownViewedMarker struct {
	markFunc    func(ctx context.Context, id int64) error
	getOrgFunc  func(ctx context.Context, id int64) (int64, error)
	parentFunc  func(ctx context.Context, orgID int64) (*int64, error)
	markCalls   int
}

func (m *mockShutdownViewedMarker) MarkShutdownAsViewed(ctx context.Context, id int64) error {
	m.markCalls++
	if m.markFunc != nil {
		return m.markFunc(ctx, id)
	}
	return nil
}

func (m *mockShutdownViewedMarker) GetShutdownOrganizationID(ctx context.Context, id int64) (int64, error) {
	if m.getOrgFunc != nil {
		return m.getOrgFunc(ctx, id)
	}
	return 1, nil
}

func (m *mockShutdownViewedMarker) GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error) {
	if m.parentFunc != nil {
		return m.parentFunc(ctx, orgID)
	}
	return nil, nil
}

func serveMarkViewed(t *testing.T, mock *mockShutdownViewedMarker, id string, claims *token.Claims) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, "/shutdowns/"+id+"/viewed", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if claims != nil {
		ctx = mwauth.ContextWithClaims(ctx, claims)
	}
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := MarkViewed(logger, mock)
	handler.ServeHTTP(rr, req)
	return rr
}

func TestMarkViewed_ScUser_AnyShutdown_OK(t *testing.T) {
	mock := &mockShutdownViewedMarker{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 99, nil },
	}
	rr := serveMarkViewed(t, mock, "7", &token.Claims{UserID: 1, Roles: []string{"sc"}})

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.markCalls != 1 {
		t.Errorf("MarkShutdownAsViewed calls = %d, want 1", mock.markCalls)
	}
}

func TestMarkViewed_CascadeUser_OwnCascade_OK(t *testing.T) {
	parent := int64(5)
	mock := &mockShutdownViewedMarker{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 10, nil },
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) { return &parent, nil },
	}
	rr := serveMarkViewed(t, mock, "7", &token.Claims{UserID: 1, OrganizationID: 5, Roles: []string{"cascade"}})

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200. body=%s", rr.Code, rr.Body.String())
	}
	if mock.markCalls != 1 {
		t.Errorf("MarkShutdownAsViewed calls = %d, want 1", mock.markCalls)
	}
}

// Regression guard for IDOR: cascade user must NOT be able to mark a foreign
// cascade's shutdown as viewed. Response must be 404 (not 403) to avoid
// leaking the existence of the record.
func TestMarkViewed_CascadeUser_ForeignCascade_NotFound(t *testing.T) {
	foreignParent := int64(7)
	mock := &mockShutdownViewedMarker{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 20, nil },
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) { return &foreignParent, nil },
	}
	rr := serveMarkViewed(t, mock, "7", &token.Claims{UserID: 1, OrganizationID: 5, Roles: []string{"cascade"}})

	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404. body=%s", rr.Code, rr.Body.String())
	}
	if mock.markCalls != 0 {
		t.Errorf("MarkShutdownAsViewed must not be called on foreign shutdown, got %d calls", mock.markCalls)
	}
}

func TestMarkViewed_ShutdownNotFound(t *testing.T) {
	mock := &mockShutdownViewedMarker{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 0, storage.ErrNotFound },
	}
	rr := serveMarkViewed(t, mock, "999", &token.Claims{UserID: 1, Roles: []string{"sc"}})

	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404. body=%s", rr.Code, rr.Body.String())
	}
	if mock.markCalls != 0 {
		t.Errorf("MarkShutdownAsViewed must not be called when lookup fails, got %d calls", mock.markCalls)
	}
}

func TestMarkViewed_InvalidID(t *testing.T) {
	mock := &mockShutdownViewedMarker{}
	rr := serveMarkViewed(t, mock, "not-a-number", &token.Claims{UserID: 1, Roles: []string{"sc"}})

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rr.Code)
	}
	if mock.markCalls != 0 {
		t.Errorf("MarkShutdownAsViewed called on invalid id, got %d calls", mock.markCalls)
	}
}

func TestMarkViewed_MarkInternalError(t *testing.T) {
	mock := &mockShutdownViewedMarker{
		getOrgFunc: func(ctx context.Context, id int64) (int64, error) { return 1, nil },
		markFunc:   func(ctx context.Context, id int64) error { return errors.New("database down") },
	}
	rr := serveMarkViewed(t, mock, "7", &token.Claims{UserID: 1, Roles: []string{"sc"}})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", rr.Code)
	}
}
