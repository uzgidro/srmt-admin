package infraeventcategory

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraeventcategorymodel "srmt-admin/internal/lib/model/infra-event-category"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
)

// --- Mocks ---

type mockCategoryGetter struct {
	getFunc func(ctx context.Context) ([]*infraeventcategorymodel.Model, error)
}

func (m *mockCategoryGetter) GetInfraEventCategories(ctx context.Context) ([]*infraeventcategorymodel.Model, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx)
	}
	return make([]*infraeventcategorymodel.Model, 0), nil
}

type mockCategoryCreator struct {
	createFunc func(ctx context.Context, slug, displayName, label string, sortOrder int) (int64, error)
}

func (m *mockCategoryCreator) CreateInfraEventCategory(ctx context.Context, slug, displayName, label string, sortOrder int) (int64, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, slug, displayName, label, sortOrder)
	}
	return 1, nil
}

type mockCategoryUpdater struct {
	updateFunc func(ctx context.Context, id int64, slug, displayName, label string, sortOrder int) error
}

func (m *mockCategoryUpdater) UpdateInfraEventCategory(ctx context.Context, id int64, slug, displayName, label string, sortOrder int) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, slug, displayName, label, sortOrder)
	}
	return nil
}

type mockCategoryDeleter struct {
	deleteFunc func(ctx context.Context, id int64) error
}

func (m *mockCategoryDeleter) DeleteInfraEventCategory(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- GET tests ---

func TestGetCategories_Empty(t *testing.T) {
	getter := &mockCategoryGetter{}
	handler := Get(testLogger(), getter)

	req := httptest.NewRequest(http.MethodGet, "/infra-event-categories", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var result []*infraeventcategorymodel.Model
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestGetCategories_WithData(t *testing.T) {
	now := time.Now()
	getter := &mockCategoryGetter{
		getFunc: func(_ context.Context) ([]*infraeventcategorymodel.Model, error) {
			return []*infraeventcategorymodel.Model{
				{ID: 1, Slug: "video", DisplayName: "Video", Label: "Video Label", SortOrder: 1, CreatedAt: now},
				{ID: 2, Slug: "comms", DisplayName: "Comms", Label: "Comms Label", SortOrder: 2, CreatedAt: now},
			}, nil
		},
	}
	handler := Get(testLogger(), getter)

	req := httptest.NewRequest(http.MethodGet, "/infra-event-categories", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var result []*infraeventcategorymodel.Model
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

// --- CREATE tests ---

func TestCreateCategory_Success(t *testing.T) {
	creator := &mockCategoryCreator{
		createFunc: func(_ context.Context, slug, _, _ string, _ int) (int64, error) {
			return 10, nil
		},
	}
	handler := Create(testLogger(), creator)

	body, _ := json.Marshal(createRequest{
		Slug:        "test",
		DisplayName: "Test",
		Label:       "Test Label",
		SortOrder:   1,
	})

	req := httptest.NewRequest(http.MethodPost, "/infra-event-categories", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	var result createResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result.ID != 10 {
		t.Errorf("expected ID 10, got %d", result.ID)
	}
}

func TestCreateCategory_DuplicateSlug(t *testing.T) {
	creator := &mockCategoryCreator{
		createFunc: func(_ context.Context, _, _, _ string, _ int) (int64, error) {
			return 0, storage.ErrUniqueViolation
		},
	}
	handler := Create(testLogger(), creator)

	body, _ := json.Marshal(createRequest{
		Slug:        "video",
		DisplayName: "Video",
		Label:       "Video Label",
	})

	req := httptest.NewRequest(http.MethodPost, "/infra-event-categories", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateCategory_MissingRequired(t *testing.T) {
	creator := &mockCategoryCreator{}
	handler := Create(testLogger(), creator)

	body, _ := json.Marshal(map[string]string{"slug": "test"}) // missing display_name and label

	req := httptest.NewRequest(http.MethodPost, "/infra-event-categories", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

// --- UPDATE tests ---

func TestUpdateCategory_Success(t *testing.T) {
	updater := &mockCategoryUpdater{}
	handler := Update(testLogger(), updater)

	body, _ := json.Marshal(updateRequest{
		Slug:        "video_v2",
		DisplayName: "Video V2",
		Label:       "Video V2 Label",
		SortOrder:   1,
	})

	req := httptest.NewRequest(http.MethodPatch, "/infra-event-categories/1", bytes.NewBuffer(body))
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

func TestUpdateCategory_NotFound(t *testing.T) {
	updater := &mockCategoryUpdater{
		updateFunc: func(_ context.Context, _ int64, _, _, _ string, _ int) error {
			return storage.ErrNotFound
		},
	}
	handler := Update(testLogger(), updater)

	body, _ := json.Marshal(updateRequest{
		Slug:        "nope",
		DisplayName: "Nope",
		Label:       "Nope Label",
	})

	req := httptest.NewRequest(http.MethodPatch, "/infra-event-categories/9999", bytes.NewBuffer(body))
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

func TestDeleteCategory_Success(t *testing.T) {
	deleter := &mockCategoryDeleter{}
	handler := Delete(testLogger(), deleter)

	req := httptest.NewRequest(http.MethodDelete, "/infra-event-categories/1", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteCategory_NotFound(t *testing.T) {
	deleter := &mockCategoryDeleter{
		deleteFunc: func(_ context.Context, _ int64) error {
			return storage.ErrNotFound
		},
	}
	handler := Delete(testLogger(), deleter)

	req := httptest.NewRequest(http.MethodDelete, "/infra-event-categories/9999", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "9999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d; body: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteCategory_Referenced(t *testing.T) {
	deleter := &mockCategoryDeleter{
		deleteFunc: func(_ context.Context, _ int64) error {
			return storage.ErrForeignKeyViolation
		},
	}
	handler := Delete(testLogger(), deleter)

	req := httptest.NewRequest(http.MethodDelete, "/infra-event-categories/1", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d; body: %s", rr.Code, rr.Body.String())
	}
}
