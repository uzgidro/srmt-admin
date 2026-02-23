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

type mockAllStructuresGetter struct {
	getFunc func(ctx context.Context) ([]*salary.SalaryStructure, error)
}

func (m *mockAllStructuresGetter) GetAllStructures(ctx context.Context) ([]*salary.SalaryStructure, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx)
	}
	return nil, nil
}

func TestGetAllStructures(t *testing.T) {
	tests := []struct {
		name           string
		mockReturn     []*salary.SalaryStructure
		mockErr        error
		wantStatusCode int
		wantLen        int
		wantErrInBody  bool
	}{
		{
			name:           "success",
			mockReturn:     []*salary.SalaryStructure{{ID: 1, BaseSalary: 5000000}},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "empty",
			mockReturn:     []*salary.SalaryStructure{},
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
			mock := &mockAllStructuresGetter{
				getFunc: func(_ context.Context) ([]*salary.SalaryStructure, error) {
					return tt.mockReturn, tt.mockErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/hrm/salary/structures", nil)
			rr := httptest.NewRecorder()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := GetAllStructures(log, mock)
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

			var result []salary.SalaryStructure
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("result len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}
