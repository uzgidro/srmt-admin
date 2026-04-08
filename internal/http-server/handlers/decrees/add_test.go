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
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"testing"
	"time"
)

type mockDecreeAdder struct {
	addFunc           func(ctx context.Context, req dto.AddDecreeRequest, createdByID int64) (int64, error)
	linkFilesFunc     func(ctx context.Context, decreeID int64, fileIDs []int64) error
	linkDocumentsFunc func(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error
}

func (m *mockDecreeAdder) AddDecree(ctx context.Context, req dto.AddDecreeRequest, createdByID int64) (int64, error) {
	if m.addFunc != nil {
		return m.addFunc(ctx, req, createdByID)
	}
	return 1, nil
}

func (m *mockDecreeAdder) LinkDecreeFiles(ctx context.Context, decreeID int64, fileIDs []int64) error {
	if m.linkFilesFunc != nil {
		return m.linkFilesFunc(ctx, decreeID, fileIDs)
	}
	return nil
}

func (m *mockDecreeAdder) LinkDecreeDocuments(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	if m.linkDocumentsFunc != nil {
		return m.linkDocumentsFunc(ctx, decreeID, links, userID)
	}
	return nil
}

func testContextWithClaims(ctx context.Context, userID int64) context.Context {
	claims := &token.Claims{
		UserID: userID,
		Name:   "Test User",
		Roles:  []string{"admin"},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

func TestAdd(t *testing.T) {
	docDate := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		body           interface{}
		userID         int64
		mockResponse   int64
		mockError      error
		wantStatusCode int
	}{
		{
			name: "successful creation with file_ids",
			body: addRequest{
				Name:         "Test decree",
				DocumentDate: docDate,
				TypeID:       1,
				FileIDs:      []int64{42, 43},
			},
			userID:         1,
			mockResponse:   10,
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "successful creation without files",
			body: addRequest{
				Name:         "Test decree",
				DocumentDate: docDate,
				TypeID:       1,
			},
			userID:         1,
			mockResponse:   11,
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "validation error - missing name",
			body: addRequest{
				DocumentDate: docDate,
				TypeID:       1,
			},
			userID:         1,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "foreign key violation",
			body: addRequest{
				Name:         "Test",
				DocumentDate: docDate,
				TypeID:       999,
			},
			userID:         1,
			mockError:      storage.ErrForeignKeyViolation,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "internal server error",
			body: addRequest{
				Name:         "Test",
				DocumentDate: docDate,
				TypeID:       1,
			},
			userID:         1,
			mockError:      errors.New("db error"),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "invalid JSON",
			body:           "not json",
			userID:         1,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDecreeAdder{
				addFunc: func(ctx context.Context, req dto.AddDecreeRequest, createdByID int64) (int64, error) {
					if tt.mockError != nil {
						return 0, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			var bodyReader io.Reader
			if str, ok := tt.body.(string); ok {
				bodyReader = bytes.NewBufferString(str)
			} else {
				bodyBytes, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewBuffer(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/decrees", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			ctx := testContextWithClaims(req.Context(), tt.userID)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Add(logger, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("got status %d, want %d, body: %s", rr.Code, tt.wantStatusCode, rr.Body.String())
			}

			if tt.wantStatusCode == http.StatusCreated {
				var resp addResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != tt.mockResponse {
					t.Errorf("got ID %d, want %d", resp.ID, tt.mockResponse)
				}
			}
		})
	}
}

func TestAdd_NoAuth(t *testing.T) {
	mock := &mockDecreeAdder{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	body := addRequest{Name: "Test", DocumentDate: time.Now(), TypeID: 1}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/decrees", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := Add(logger, mock)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdd_MultipartRejected(t *testing.T) {
	mock := &mockDecreeAdder{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	req := httptest.NewRequest(http.MethodPost, "/decrees", bytes.NewBufferString("--boundary\r\n"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	ctx := testContextWithClaims(req.Context(), 1)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := Add(logger, mock)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("multipart should be rejected: got status %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
