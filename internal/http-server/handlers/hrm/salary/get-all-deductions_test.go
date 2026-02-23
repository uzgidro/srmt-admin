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

	salary "srmt-admin/internal/lib/model/hrm/salary"
)

type mockAllDeductionsGetter struct {
	getFunc func(ctx context.Context) ([]*salary.Deduction, error)
}

func (m *mockAllDeductionsGetter) GetAllDeductions(ctx context.Context) ([]*salary.Deduction, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx)
	}
	return nil, nil
}

func TestGetAllDeductions(t *testing.T) {
	tests := []struct {
		name           string
		mockReturn     []*salary.Deduction
		mockErr        error
		wantStatusCode int
		wantLen        int
		wantErrInBody  bool
	}{
		{
			name:           "success",
			mockReturn:     []*salary.Deduction{{ID: 1, Amount: 100000}},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "empty",
			mockReturn:     []*salary.Deduction{},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:           "service error",
			mockErr:        errors.New("db error"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAllDeductionsGetter{
				getFunc: func(_ context.Context) ([]*salary.Deduction, error) {
					return tt.mockReturn, tt.mockErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/hrm/salary/deductions", nil)
			rr := httptest.NewRecorder()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := GetAllDeductions(log, mock)
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

			var result []salary.Deduction
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("result len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}
