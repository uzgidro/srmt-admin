package infraevent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"srmt-admin/internal/lib/dto"
	infraeventmodel "srmt-admin/internal/lib/model/infra-event"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"

	"github.com/go-chi/chi/v5"
)

// --- Mocks ---

type mockEventGetter struct {
	getFunc       func(ctx context.Context, categoryID int64, day time.Time) ([]*infraeventmodel.ResponseModel, error)
	getByDateFunc func(ctx context.Context, day time.Time) ([]*infraeventmodel.ResponseModel, error)
}

func (m *mockEventGetter) GetInfraEvents(ctx context.Context, categoryID int64, day time.Time) ([]*infraeventmodel.ResponseModel, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, categoryID, day)
	}
	return make([]*infraeventmodel.ResponseModel, 0), nil
}

func (m *mockEventGetter) GetInfraEventsByDate(ctx context.Context, day time.Time) ([]*infraeventmodel.ResponseModel, error) {
	if m.getByDateFunc != nil {
		return m.getByDateFunc(ctx, day)
	}
	return make([]*infraeventmodel.ResponseModel, 0), nil
}

type mockEventAdder struct {
	createFunc func(ctx context.Context, req dto.AddInfraEventRequest) (int64, error)
	linkFunc   func(ctx context.Context, eventID int64, fileIDs []int64) error
}

func (m *mockEventAdder) CreateInfraEvent(ctx context.Context, req dto.AddInfraEventRequest) (int64, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return 1, nil
}

func (m *mockEventAdder) LinkInfraEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error {
	if m.linkFunc != nil {
		return m.linkFunc(ctx, eventID, fileIDs)
	}
	return nil
}

type mockEventEditor struct {
	updateFunc func(ctx context.Context, id int64, req dto.EditInfraEventRequest) error
	unlinkFunc func(ctx context.Context, eventID int64) error
	linkFunc   func(ctx context.Context, eventID int64, fileIDs []int64) error
}

func (m *mockEventEditor) UpdateInfraEvent(ctx context.Context, id int64, req dto.EditInfraEventRequest) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, req)
	}
	return nil
}

func (m *mockEventEditor) UnlinkInfraEventFiles(ctx context.Context, eventID int64) error {
	if m.unlinkFunc != nil {
		return m.unlinkFunc(ctx, eventID)
	}
	return nil
}

func (m *mockEventEditor) LinkInfraEventFiles(ctx context.Context, eventID int64, fileIDs []int64) error {
	if m.linkFunc != nil {
		return m.linkFunc(ctx, eventID, fileIDs)
	}
	return nil
}

type mockEventDeleter struct {
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockEventDeleter) DeleteInfraEvent(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

// mockMinioURLGenerator is a no-op minio URL generator for tests
type mockMinioURLGenerator struct{}

func (m *mockMinioURLGenerator) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (*url.URL, error) {
	u, _ := url.Parse("http://test/file.pdf")
	return u, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func contextWithClaims(ctx context.Context, userID int64) context.Context {
	claims := &token.Claims{
		UserID: userID,
		Name:   "Test User",
		Roles:  []string{"sc"},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

// --- GET tests ---

func TestGetEvents_Empty(t *testing.T) {
	getter := &mockEventGetter{}
	loc := time.UTC
	handler := Get(testLogger(), getter, &mockMinioURLGenerator{}, loc)

	req := httptest.NewRequest(http.MethodGet, "/infra-events?date=2026-03-28", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var result []*infraeventmodel.ResponseWithURLs
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestGetEvents_FilterByCategoryAndDate(t *testing.T) {
	now := time.Now()
	getter := &mockEventGetter{
		getFunc: func(_ context.Context, categoryID int64, _ time.Time) ([]*infraeventmodel.ResponseModel, error) {
			if categoryID != 1 {
				return make([]*infraeventmodel.ResponseModel, 0), nil
			}
			return []*infraeventmodel.ResponseModel{
				{ID: 1, CategoryID: 1, CategorySlug: "video", CategoryName: "Video", Description: "Camera down", OccurredAt: now, CreatedAt: now},
			}, nil
		},
	}
	loc := time.UTC
	handler := Get(testLogger(), getter, &mockMinioURLGenerator{}, loc)

	req := httptest.NewRequest(http.MethodGet, "/infra-events?category_id=1&date=2026-03-28", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var result []*infraeventmodel.ResponseWithURLs
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

// --- CREATE tests ---

func TestCreateEvent_Success(t *testing.T) {
	adder := &mockEventAdder{
		createFunc: func(_ context.Context, _ dto.AddInfraEventRequest) (int64, error) {
			return 42, nil
		},
	}
	handler := Create(testLogger(), adder)

	body, _ := json.Marshal(addRequest{
		CategoryID:     1,
		OrganizationID: 1,
		OccurredAt:     time.Now().Format(time.RFC3339),
		Description:    "Camera offline",
	})

	req := httptest.NewRequest(http.MethodPost, "/infra-events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithClaims(req.Context(), 1))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var result addResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result.ID != 42 {
		t.Errorf("expected ID 42, got %d", result.ID)
	}
}

func TestCreateEvent_InvalidCategory(t *testing.T) {
	adder := &mockEventAdder{
		createFunc: func(_ context.Context, _ dto.AddInfraEventRequest) (int64, error) {
			return 0, storage.ErrForeignKeyViolation
		},
	}
	handler := Create(testLogger(), adder)

	body, _ := json.Marshal(addRequest{
		CategoryID:     9999,
		OrganizationID: 1,
		OccurredAt:     time.Now().Format(time.RFC3339),
		Description:    "Test",
	})

	req := httptest.NewRequest(http.MethodPost, "/infra-events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithClaims(req.Context(), 1))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateEvent_Unauthorized(t *testing.T) {
	adder := &mockEventAdder{}
	handler := Create(testLogger(), adder)

	body, _ := json.Marshal(addRequest{
		CategoryID:     1,
		OrganizationID: 1,
		OccurredAt:     time.Now().Format(time.RFC3339),
		Description:    "Test",
	})

	req := httptest.NewRequest(http.MethodPost, "/infra-events", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No claims in context

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// --- UPDATE tests ---

func TestUpdateEvent_Success(t *testing.T) {
	editor := &mockEventEditor{}
	handler := Update(testLogger(), editor)

	desc := "Updated description"
	body, _ := json.Marshal(editRequest{Description: &desc})

	req := httptest.NewRequest(http.MethodPatch, "/infra-events/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateEvent_NotFound(t *testing.T) {
	editor := &mockEventEditor{
		updateFunc: func(_ context.Context, _ int64, _ dto.EditInfraEventRequest) error {
			return storage.ErrNotFound
		},
	}
	handler := Update(testLogger(), editor)

	desc := "Updated"
	body, _ := json.Marshal(editRequest{Description: &desc})

	req := httptest.NewRequest(http.MethodPatch, "/infra-events/9999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "9999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// --- DELETE tests ---

func TestDeleteEvent_Success(t *testing.T) {
	deleter := &mockEventDeleter{}
	handler := Delete(testLogger(), deleter)

	req := httptest.NewRequest(http.MethodDelete, "/infra-events/1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteEvent_NotFound(t *testing.T) {
	deleter := &mockEventDeleter{
		deleteFunc: func(_ context.Context, _ int64) error {
			return storage.ErrNotFound
		},
	}
	handler := Delete(testLogger(), deleter)

	req := httptest.NewRequest(http.MethodDelete, "/infra-events/9999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "9999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
}
