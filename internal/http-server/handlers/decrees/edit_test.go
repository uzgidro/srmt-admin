package decrees

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
)

type mockDecreeEditor struct {
	editFunc           func(ctx context.Context, id int64, req dto.EditDecreeRequest, updatedByID int64) error
	unlinkFilesFunc    func(ctx context.Context, decreeID int64) error
	linkFilesFunc      func(ctx context.Context, decreeID int64, fileIDs []int64) error
	unlinkDocsFunc     func(ctx context.Context, decreeID int64) error
	linkDocsFunc       func(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error
	unlinkFilesCalled  bool
	linkFilesCalled    bool
	linkedFileIDs      []int64
}

func (m *mockDecreeEditor) EditDecree(ctx context.Context, id int64, req dto.EditDecreeRequest, updatedByID int64) error {
	if m.editFunc != nil {
		return m.editFunc(ctx, id, req, updatedByID)
	}
	return nil
}

func (m *mockDecreeEditor) UnlinkDecreeFiles(ctx context.Context, decreeID int64) error {
	m.unlinkFilesCalled = true
	if m.unlinkFilesFunc != nil {
		return m.unlinkFilesFunc(ctx, decreeID)
	}
	return nil
}

func (m *mockDecreeEditor) LinkDecreeFiles(ctx context.Context, decreeID int64, fileIDs []int64) error {
	m.linkFilesCalled = true
	m.linkedFileIDs = fileIDs
	if m.linkFilesFunc != nil {
		return m.linkFilesFunc(ctx, decreeID, fileIDs)
	}
	return nil
}

func (m *mockDecreeEditor) UnlinkDecreeDocuments(ctx context.Context, decreeID int64) error {
	if m.unlinkDocsFunc != nil {
		return m.unlinkDocsFunc(ctx, decreeID)
	}
	return nil
}

func (m *mockDecreeEditor) LinkDecreeDocuments(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	if m.linkDocsFunc != nil {
		return m.linkDocsFunc(ctx, decreeID, links, userID)
	}
	return nil
}

func newEditRequest(t *testing.T, id string, body interface{}) *http.Request {
	t.Helper()
	var bodyReader io.Reader
	if str, ok := body.(string); ok {
		bodyReader = bytes.NewBufferString(str)
	} else {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(bodyBytes)
	}
	req := httptest.NewRequest(http.MethodPatch, "/decrees/"+id, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	ctx := testContextWithClaims(req.Context(), 1)
	return req.WithContext(ctx)
}

func TestEdit(t *testing.T) {
	tests := []struct {
		name              string
		id                string
		body              interface{}
		mockError         error
		wantStatusCode    int
		wantUnlinkFiles   bool
		wantLinkFiles     bool
		wantLinkedFileIDs []int64
	}{
		{
			name:           "successful edit with name",
			id:             "1",
			body:           map[string]interface{}{"name": "Updated"},
			wantStatusCode: http.StatusOK,
		},
		{
			name:            "file_ids present replaces files",
			id:              "1",
			body:            map[string]interface{}{"file_ids": []int64{10, 20}},
			wantStatusCode:  http.StatusOK,
			wantUnlinkFiles: true,
			wantLinkFiles:   true,
			wantLinkedFileIDs: []int64{10, 20},
		},
		{
			name:            "file_ids empty clears files",
			id:              "1",
			body:            map[string]interface{}{"file_ids": []int64{}},
			wantStatusCode:  http.StatusOK,
			wantUnlinkFiles: true,
			wantLinkFiles:   false,
		},
		{
			name:           "file_ids omitted does not touch files",
			id:             "1",
			body:           map[string]interface{}{"name": "No file changes"},
			wantStatusCode: http.StatusOK,
			wantUnlinkFiles: false,
			wantLinkFiles:   false,
		},
		{
			name:           "invalid id parameter",
			id:             "abc",
			body:           map[string]interface{}{"name": "Test"},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "decree not found",
			id:             "999",
			body:           map[string]interface{}{"name": "Test"},
			mockError:      storage.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "foreign key violation",
			id:             "1",
			body:           map[string]interface{}{"type_id": 999},
			mockError:      storage.ErrForeignKeyViolation,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "internal server error",
			id:             "1",
			body:           map[string]interface{}{"name": "Test"},
			mockError:      errors.New("db error"),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "invalid JSON",
			id:             "1",
			body:           "not json",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDecreeEditor{
				editFunc: func(ctx context.Context, id int64, req dto.EditDecreeRequest, updatedByID int64) error {
					if tt.mockError != nil {
						return tt.mockError
					}
					return nil
				},
			}

			req := newEditRequest(t, tt.id, tt.body)
			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Edit(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("got status %d, want %d, body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			if tt.wantUnlinkFiles != mock.unlinkFilesCalled {
				t.Errorf("unlinkFiles called=%v, want %v", mock.unlinkFilesCalled, tt.wantUnlinkFiles)
			}
			if tt.wantLinkFiles != mock.linkFilesCalled {
				t.Errorf("linkFiles called=%v, want %v", mock.linkFilesCalled, tt.wantLinkFiles)
			}
			if tt.wantLinkedFileIDs != nil {
				if len(mock.linkedFileIDs) != len(tt.wantLinkedFileIDs) {
					t.Errorf("linked file IDs = %v, want %v", mock.linkedFileIDs, tt.wantLinkedFileIDs)
				}
			}
		})
	}
}

func TestEdit_NoAuth(t *testing.T) {
	mock := &mockDecreeEditor{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	body, _ := json.Marshal(map[string]interface{}{"name": "Test"})
	req := httptest.NewRequest(http.MethodPatch, "/decrees/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler := Edit(logger, mock)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestEdit_MultipartRejected(t *testing.T) {
	mock := &mockDecreeEditor{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	req := httptest.NewRequest(http.MethodPatch, "/decrees/1", bytes.NewBufferString("--boundary\r\n"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	ctx := testContextWithClaims(req.Context(), 1)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler := Edit(logger, mock)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("multipart should be rejected: got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
