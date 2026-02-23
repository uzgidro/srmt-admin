package salary

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/dto"
	salary "srmt-admin/internal/lib/model/hrm/salary"
	"srmt-admin/internal/token"
)

type mockSalaryGetter struct {
	getAllFunc     func(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error)
	capturedFilter dto.SalaryFilters
}

func (m *mockSalaryGetter) GetAll(ctx context.Context, filters dto.SalaryFilters) ([]*salary.Salary, error) {
	m.capturedFilter = filters
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx, filters)
	}
	return nil, nil
}

func contextWithClaims(ctx context.Context, contactID int64) context.Context {
	claims := &token.Claims{
		ContactID: contactID,
		Name:      "Test User",
		Roles:     []string{"hrm_employee"},
	}
	return mwauth.ContextWithClaims(ctx, claims)
}

func TestGet(t *testing.T) {
	tests := []struct {
		name           string
		contactID      int64
		withClaims     bool
		mockReturn     []*salary.Salary
		mockErr        error
		wantStatusCode int
		wantLen        int
		wantErrInBody  bool
	}{
		{
			name:           "success — returns salary list",
			contactID:      42,
			withClaims:     true,
			mockReturn:     []*salary.Salary{{ID: 1}},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "success — empty result",
			contactID:      42,
			withClaims:     true,
			mockReturn:     []*salary.Salary{},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:           "unauthorized — no claims",
			withClaims:     false,
			wantStatusCode: http.StatusUnauthorized,
			wantErrInBody:  true,
		},
		{
			name:           "service error",
			contactID:      42,
			withClaims:     true,
			mockErr:        errors.New("db error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSalaryGetter{
				getAllFunc: func(_ context.Context, _ dto.SalaryFilters) ([]*salary.Salary, error) {
					return tt.mockReturn, tt.mockErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/my/salary", nil)
			if tt.withClaims {
				req = req.WithContext(contextWithClaims(req.Context(), tt.contactID))
			}

			rr := httptest.NewRecorder()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Get(log, mock)
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatusCode)
			}

			if tt.wantErrInBody {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("expected error in body, got: %v", resp)
				}
				return
			}

			var result []salary.Salary
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("result len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestGet_FiltersPassthrough(t *testing.T) {
	mock := &mockSalaryGetter{
		getAllFunc: func(_ context.Context, _ dto.SalaryFilters) ([]*salary.Salary, error) {
			return []*salary.Salary{}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/salary", nil)
	req = req.WithContext(contextWithClaims(req.Context(), 42))

	rr := httptest.NewRecorder()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := Get(log, mock)
	handler.ServeHTTP(rr, req)

	if mock.capturedFilter.EmployeeID == nil {
		t.Fatal("expected EmployeeID filter to be set")
	}
	if *mock.capturedFilter.EmployeeID != 42 {
		t.Errorf("EmployeeID filter = %d, want 42", *mock.capturedFilter.EmployeeID)
	}
}
