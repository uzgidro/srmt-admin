package competencies

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
	"srmt-admin/internal/lib/model/hrm/competency"
	"srmt-admin/internal/token"
)

type mockCompetencyGetter struct {
	getFunc            func(ctx context.Context, employeeID int64) ([]*competency.AssessmentSession, error)
	capturedEmployeeID int64
}

func (m *mockCompetencyGetter) GetEmployeeAssessments(ctx context.Context, employeeID int64) ([]*competency.AssessmentSession, error) {
	m.capturedEmployeeID = employeeID
	if m.getFunc != nil {
		return m.getFunc(ctx, employeeID)
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
		mockReturn     []*competency.AssessmentSession
		mockErr        error
		wantStatusCode int
		wantLen        int
		wantErrInBody  bool
	}{
		{
			name:           "success",
			contactID:      42,
			withClaims:     true,
			mockReturn:     []*competency.AssessmentSession{{ID: 1}},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "empty",
			contactID:      42,
			withClaims:     true,
			mockReturn:     []*competency.AssessmentSession{},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:           "unauthorized",
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
			mock := &mockCompetencyGetter{
				getFunc: func(_ context.Context, _ int64) ([]*competency.AssessmentSession, error) {
					return tt.mockReturn, tt.mockErr
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/my/competencies", nil)
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

			var result []competency.AssessmentSession
			if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("result len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestGet_EmployeeIDPassthrough(t *testing.T) {
	mock := &mockCompetencyGetter{
		getFunc: func(_ context.Context, _ int64) ([]*competency.AssessmentSession, error) {
			return []*competency.AssessmentSession{}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/my/competencies", nil)
	req = req.WithContext(contextWithClaims(req.Context(), 42))

	rr := httptest.NewRecorder()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := Get(log, mock)
	handler.ServeHTTP(rr, req)

	if mock.capturedEmployeeID != 42 {
		t.Errorf("employeeID = %d, want 42", mock.capturedEmployeeID)
	}
}
