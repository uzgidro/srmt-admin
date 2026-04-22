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

	"github.com/go-chi/chi/v5"
)

// mockShutdownGetter implements the ShutdownGetter interface with the new
// GetOrganizationParentID method (to be added to the production interface so
// cascade RBAC can run inside the handler).
type mockShutdownGetter struct {
	getFunc      func(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error)
	parentFunc   func(ctx context.Context, orgID int64) (*int64, error)
	getCalls     int
}

func (m *mockShutdownGetter) GetShutdownsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error) {
	m.getCalls++
	if m.getFunc != nil {
		return m.getFunc(ctx, orgID, startDate, endDate)
	}
	return []*shutdown.ResponseModel{}, nil
}

func (m *mockShutdownGetter) GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error) {
	if m.parentFunc != nil {
		return m.parentFunc(ctx, orgID)
	}
	return nil, nil
}

// mockMinioRepo is a minimal MinioURLGenerator for tests; it returns a stub URL.
type mockMinioRepo struct{}

func (m *mockMinioRepo) GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error) {
	return url.Parse("https://minio.example.com/bucket/" + objectName)
}

func int64Ptr(i int64) *int64 {
	return &i
}

func ctxWithClaims(userID, orgID int64, role string) context.Context {
	claims := &token.Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Name:           "Test User",
		Roles:          []string{role},
	}
	return mwauth.ContextWithClaims(context.Background(), claims)
}

func runGetWithCtx(t *testing.T, mock *mockShutdownGetter, ctx context.Context, gesID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/ges/"+gesID+"/shutdowns", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", gesID)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(logger, mock, &mockMinioRepo{}, time.UTC)
	handler.ServeHTTP(rr, req)
	return rr
}

func TestGet_CascadeUser_OwnGES_OK(t *testing.T) {
	mock := &mockShutdownGetter{
		getFunc: func(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error) {
			return []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: orgID, OrganizationName: "GES 10"},
			}, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 10 {
				return int64Ptr(5), nil
			}
			return nil, nil
		},
	}

	ctx := ctxWithClaims(1, 5, "cascade")
	rr := runGetWithCtx(t, mock, ctx, "10")

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal body: %v; raw: %s", err, rr.Body.String())
	}
	if len(result) != 1 {
		t.Errorf("expected 1 shutdown in response, got %d", len(result))
	}
}

func TestGet_CascadeUser_ForeignGES_NotFound(t *testing.T) {
	mock := &mockShutdownGetter{
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			if orgID == 20 {
				return int64Ptr(7), nil
			}
			return nil, nil
		},
	}

	ctx := ctxWithClaims(1, 5, "cascade")
	rr := runGetWithCtx(t, mock, ctx, "20")

	// Foreign GES must be indistinguishable from a missing GES.
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.getCalls != 0 {
		t.Errorf("GetShutdownsByOrgID must NOT be called for foreign GES, called %d times", mock.getCalls)
	}
}

func TestGet_ScUser_AnyGES_OK(t *testing.T) {
	parentCalled := false
	mock := &mockShutdownGetter{
		getFunc: func(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error) {
			return []*shutdown.ResponseModel{}, nil
		},
		parentFunc: func(ctx context.Context, orgID int64) (*int64, error) {
			parentCalled = true
			return int64Ptr(7), nil
		},
	}

	ctx := ctxWithClaims(1, 5, "sc")
	rr := runGetWithCtx(t, mock, ctx, "20")

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if parentCalled {
		t.Error("parent lookup must NOT be called for sc role")
	}
}

func TestGet_NonCascadeNonSc_OK_IfOwnOrg(t *testing.T) {
	mock := &mockShutdownGetter{
		getFunc: func(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error) {
			return []*shutdown.ResponseModel{}, nil
		},
	}

	ctx := ctxWithClaims(1, 10, "reservoir")
	rr := runGetWithCtx(t, mock, ctx, "10")

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGet_NonCascadeNonSc_NotFound_IfForeignOrg(t *testing.T) {
	mock := &mockShutdownGetter{}

	ctx := ctxWithClaims(1, 10, "reservoir")
	rr := runGetWithCtx(t, mock, ctx, "99")

	// Foreign org — 404 (enumeration protection) rather than 403.
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 (enumeration protection), got %d: %s", rr.Code, rr.Body.String())
	}
	if mock.getCalls != 0 {
		t.Errorf("GetShutdownsByOrgID must NOT be called for foreign org, called %d times", mock.getCalls)
	}
}
