package organizations

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
)

type mockSetter struct {
	gotUserID  int64
	gotOrgIDs  []int64
	called     bool
	returnErr  error
}

func (m *mockSetter) SetUserOrganizations(_ context.Context, userID int64, orgIDs []int64) error {
	m.called = true
	m.gotUserID = userID
	m.gotOrgIDs = orgIDs
	return m.returnErr
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func doRequest(t *testing.T, setter *mockSetter, userIDPath, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Put("/users/{userID}/organizations", New(discardLogger(), setter))

	var rdr *bytes.Buffer
	if body == "" {
		rdr = bytes.NewBuffer(nil)
	} else {
		rdr = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(http.MethodPut, "/users/"+userIDPath+"/organizations", rdr)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestSetOrganizations_OK(t *testing.T) {
	setter := &mockSetter{}
	rr := doRequest(t, setter, "42", `{"organization_ids": [5, 10]}`)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: want 204, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if !setter.called {
		t.Fatal("repo SetUserOrganizations must be called")
	}
	if setter.gotUserID != 42 {
		t.Errorf("userID: want 42, got %d", setter.gotUserID)
	}
	if len(setter.gotOrgIDs) != 2 || setter.gotOrgIDs[0] != 5 || setter.gotOrgIDs[1] != 10 {
		t.Errorf("orgIDs: want [5 10], got %v", setter.gotOrgIDs)
	}
}

// An empty list is valid — it clears all bindings.
func TestSetOrganizations_EmptyList_OK(t *testing.T) {
	setter := &mockSetter{}
	rr := doRequest(t, setter, "42", `{"organization_ids": []}`)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status: want 204, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if !setter.called {
		t.Fatal("repo must be called for empty list (clears bindings)")
	}
	if len(setter.gotOrgIDs) != 0 {
		t.Errorf("orgIDs: want empty, got %v", setter.gotOrgIDs)
	}
}

// A missing organization_ids field is rejected — nil != empty list.
func TestSetOrganizations_MissingField_BadRequest(t *testing.T) {
	setter := &mockSetter{}
	rr := doRequest(t, setter, "42", `{}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", rr.Code)
	}
	if setter.called {
		t.Error("repo MUST NOT be called when organization_ids is absent")
	}
}

func TestSetOrganizations_InvalidUserID_BadRequest(t *testing.T) {
	setter := &mockSetter{}
	rr := doRequest(t, setter, "not-a-number", `{"organization_ids": [5]}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", rr.Code)
	}
	if setter.called {
		t.Error("repo MUST NOT be called on invalid user id")
	}
}

func TestSetOrganizations_ForeignKeyViolation_BadRequest(t *testing.T) {
	setter := &mockSetter{returnErr: storage.ErrForeignKeyViolation}
	rr := doRequest(t, setter, "42", `{"organization_ids": [999]}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400 (unknown user/org), got %d", rr.Code)
	}
}

func TestSetOrganizations_RepoError_InternalError(t *testing.T) {
	setter := &mockSetter{returnErr: storage.ErrNotFound}
	rr := doRequest(t, setter, "42", `{"organization_ids": [5]}`)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d", rr.Code)
	}
}
