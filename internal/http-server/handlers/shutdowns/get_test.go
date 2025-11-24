package shutdowns

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"srmt-admin/internal/lib/model/shutdown"
	"testing"
	"time"
)

// mockShutdownGetter is a mock implementation of shutdownGetter interface
type mockShutdownGetter struct {
	getShutdownsFunc   func(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
	getOrgTypesMapFunc func(ctx context.Context) (map[int64][]string, error)
}

func (m *mockShutdownGetter) GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
	if m.getShutdownsFunc != nil {
		return m.getShutdownsFunc(ctx, day)
	}
	return []*shutdown.ResponseModel{}, nil
}

func (m *mockShutdownGetter) GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error) {
	if m.getOrgTypesMapFunc != nil {
		return m.getOrgTypesMapFunc(ctx)
	}
	return map[int64][]string{}, nil
}

func TestGet(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Now().In(loc)

	tests := []struct {
		name           string
		queryDate      string
		mockShutdowns  []*shutdown.ResponseModel
		mockOrgTypes   map[int64][]string
		mockError      error
		mockOrgError   error
		wantStatusCode int
		wantErrInBody  bool
		description    string
	}{
		{
			name:      "successful get with no date (should use today)",
			queryDate: "",
			mockShutdowns: []*shutdown.ResponseModel{
				{
					ID:               1,
					OrganizationID:   1,
					OrganizationName: "Test GES",
					StartedAt:        now,
					CreatedByUserFIO: "John Doe",
					CreatedByUserID:  1,
					CreatedAt:        now,
				},
			},
			mockOrgTypes: map[int64][]string{
				1: {"ges"},
			},
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should return shutdowns for today when no date provided",
		},
		{
			name:      "successful get with specific date",
			queryDate: "2024-01-15",
			mockShutdowns: []*shutdown.ResponseModel{
				{
					ID:               1,
					OrganizationID:   1,
					OrganizationName: "Test Mini",
					StartedAt:        time.Date(2024, 1, 15, 10, 0, 0, 0, loc),
					CreatedByUserFIO: "Jane Doe",
					CreatedByUserID:  2,
					CreatedAt:        time.Date(2024, 1, 15, 10, 0, 0, 0, loc),
				},
			},
			mockOrgTypes: map[int64][]string{
				1: {"mini"},
			},
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should return shutdowns for specific date",
		},
		{
			name:           "successful get with empty results",
			queryDate:      "2020-01-01",
			mockShutdowns:  []*shutdown.ResponseModel{},
			mockOrgTypes:   map[int64][]string{},
			wantStatusCode: http.StatusOK,
			wantErrInBody:  false,
			description:    "Should return empty groups when no shutdowns found",
		},
		{
			name:           "error - invalid date format",
			queryDate:      "invalid-date",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
			description:    "Should return bad request for invalid date format",
		},
		{
			name:           "error - wrong date format (MM-DD-YYYY)",
			queryDate:      "01-15-2024",
			wantStatusCode: http.StatusBadRequest,
			wantErrInBody:  true,
			description:    "Should return bad request for wrong date format",
		},
		{
			name:           "error - GetShutdowns fails",
			queryDate:      "2024-01-15",
			mockError:      errors.New("database connection failed"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
			description:    "Should handle GetShutdowns error gracefully",
		},
		{
			name:      "error - GetOrganizationTypesMap fails",
			queryDate: "2024-01-15",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "Test", StartedAt: now, CreatedByUserFIO: "Test", CreatedByUserID: 1, CreatedAt: now},
			},
			mockOrgError:   errors.New("failed to get org types"),
			wantStatusCode: http.StatusInternalServerError,
			wantErrInBody:  true,
			description:    "Should handle GetOrganizationTypesMap error gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock getter
			mock := &mockShutdownGetter{
				getShutdownsFunc: func(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockShutdowns, nil
				},
				getOrgTypesMapFunc: func(ctx context.Context) (map[int64][]string, error) {
					if tt.mockOrgError != nil {
						return nil, tt.mockOrgError
					}
					return tt.mockOrgTypes, nil
				},
			}

			// Create request
			url := "/shutdowns"
			if tt.queryDate != "" {
				url += "?date=" + tt.queryDate
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Create logger
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			// Call handler
			handler := Get(logger, mock, loc)
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.wantStatusCode {
				t.Errorf("%s: handler returned wrong status code: got %v want %v",
					tt.description, rr.Code, tt.wantStatusCode)
			}

			// Check if error is in body when expected
			if tt.wantErrInBody {
				var resp map[string]interface{}
				json.Unmarshal(rr.Body.Bytes(), &resp)
				if resp["error"] == nil || resp["error"] == "" {
					t.Errorf("%s: expected error in response body, got: %v",
						tt.description, resp)
				}
			}

			// For successful cases, verify grouped response structure
			if tt.wantStatusCode == http.StatusOK && !tt.wantErrInBody {
				var resp shutdown.GroupedResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				if err != nil {
					t.Errorf("%s: failed to unmarshal response: %v", tt.description, err)
				}

				// Verify response has required fields (Note: Other may be nil due to omitempty tag)
				if resp.Ges == nil || resp.Mini == nil || resp.Micro == nil {
					t.Errorf("%s: response missing required fields", tt.description)
				}
			}
		})
	}
}

// TestGet_Grouping tests the organization type grouping logic
func TestGet_Grouping(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Now().In(loc)

	tests := []struct {
		name          string
		mockShutdowns []*shutdown.ResponseModel
		mockOrgTypes  map[int64][]string
		expectedGes   int
		expectedMini  int
		expectedMicro int
		expectedOther int
		description   string
	}{
		{
			name: "group shutdowns by GES",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "GES Org 1", StartedAt: now, CreatedByUserFIO: "User 1", CreatedByUserID: 1, CreatedAt: now},
				{ID: 2, OrganizationID: 2, OrganizationName: "GES Org 2", StartedAt: now, CreatedByUserFIO: "User 2", CreatedByUserID: 2, CreatedAt: now},
			},
			mockOrgTypes: map[int64][]string{
				1: {"ges"},
				2: {"ges"},
			},
			expectedGes:   2,
			expectedMini:  0,
			expectedMicro: 0,
			expectedOther: 0,
			description:   "Should group all GES organizations together",
		},
		{
			name: "group shutdowns by Mini",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "Mini Org 1", StartedAt: now, CreatedByUserFIO: "User 1", CreatedByUserID: 1, CreatedAt: now},
				{ID: 2, OrganizationID: 2, OrganizationName: "Mini Org 2", StartedAt: now, CreatedByUserFIO: "User 2", CreatedByUserID: 2, CreatedAt: now},
			},
			mockOrgTypes: map[int64][]string{
				1: {"mini"},
				2: {"mini"},
			},
			expectedGes:   0,
			expectedMini:  2,
			expectedMicro: 0,
			expectedOther: 0,
			description:   "Should group all Mini organizations together",
		},
		{
			name: "group shutdowns by Micro",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "Micro Org", StartedAt: now, CreatedByUserFIO: "User 1", CreatedByUserID: 1, CreatedAt: now},
			},
			mockOrgTypes: map[int64][]string{
				1: {"micro"},
			},
			expectedGes:   0,
			expectedMini:  0,
			expectedMicro: 1,
			expectedOther: 0,
			description:   "Should group Micro organizations",
		},
		{
			name: "mixed organization types",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "GES", StartedAt: now, CreatedByUserFIO: "User 1", CreatedByUserID: 1, CreatedAt: now},
				{ID: 2, OrganizationID: 2, OrganizationName: "Mini", StartedAt: now, CreatedByUserFIO: "User 2", CreatedByUserID: 2, CreatedAt: now},
				{ID: 3, OrganizationID: 3, OrganizationName: "Micro", StartedAt: now, CreatedByUserFIO: "User 3", CreatedByUserID: 3, CreatedAt: now},
			},
			mockOrgTypes: map[int64][]string{
				1: {"ges"},
				2: {"mini"},
				3: {"micro"},
			},
			expectedGes:   1,
			expectedMini:  1,
			expectedMicro: 1,
			expectedOther: 0,
			description:   "Should correctly group mixed organization types",
		},
		{
			name: "organization without type goes to Other",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "Unknown Org", StartedAt: now, CreatedByUserFIO: "User 1", CreatedByUserID: 1, CreatedAt: now},
			},
			mockOrgTypes:  map[int64][]string{},
			expectedGes:   0,
			expectedMini:  0,
			expectedMicro: 0,
			expectedOther: 1,
			description:   "Should put organizations without type in Other",
		},
		{
			name: "organization with unknown type goes to Other",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "Unknown Type", StartedAt: now, CreatedByUserFIO: "User 1", CreatedByUserID: 1, CreatedAt: now},
			},
			mockOrgTypes: map[int64][]string{
				1: {"unknown", "other-type"},
			},
			expectedGes:   0,
			expectedMini:  0,
			expectedMicro: 0,
			expectedOther: 1,
			description:   "Should put organizations with unknown types in Other",
		},
		{
			name: "organization with multiple types (including ges and mini)",
			mockShutdowns: []*shutdown.ResponseModel{
				{ID: 1, OrganizationID: 1, OrganizationName: "Multi Type", StartedAt: now, CreatedByUserFIO: "User 1", CreatedByUserID: 1, CreatedAt: now},
			},
			mockOrgTypes: map[int64][]string{
				1: {"ges", "mini"},
			},
			expectedGes:   1,
			expectedMini:  1, // Note: The shutdown appears in BOTH groups based on the handler logic
			expectedMicro: 0,
			expectedOther: 0,
			description:   "Should add shutdown to all matching type groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockShutdownGetter{
				getShutdownsFunc: func(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
					return tt.mockShutdowns, nil
				},
				getOrgTypesMapFunc: func(ctx context.Context) (map[int64][]string, error) {
					return tt.mockOrgTypes, nil
				},
			}

			req := httptest.NewRequest(http.MethodGet, "/shutdowns", nil)
			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Get(logger, mock, loc)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %v", tt.description, rr.Code)
			}

			var resp shutdown.GroupedResponse
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			if err != nil {
				t.Errorf("%s: failed to unmarshal response: %v", tt.description, err)
			}

			if len(resp.Ges) != tt.expectedGes {
				t.Errorf("%s: GES count = %v, want %v", tt.description, len(resp.Ges), tt.expectedGes)
			}
			if len(resp.Mini) != tt.expectedMini {
				t.Errorf("%s: Mini count = %v, want %v", tt.description, len(resp.Mini), tt.expectedMini)
			}
			if len(resp.Micro) != tt.expectedMicro {
				t.Errorf("%s: Micro count = %v, want %v", tt.description, len(resp.Micro), tt.expectedMicro)
			}
			if len(resp.Other) != tt.expectedOther {
				t.Errorf("%s: Other count = %v, want %v", tt.description, len(resp.Other), tt.expectedOther)
			}

			t.Logf("%s: PASSED - GES:%d, Mini:%d, Micro:%d, Other:%d",
				tt.description, len(resp.Ges), len(resp.Mini), len(resp.Micro), len(resp.Other))
		})
	}
}

// TestGet_DateParsing tests various date parsing scenarios
func TestGet_DateParsing(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")

	tests := []struct {
		name        string
		queryDate   string
		expectError bool
		description string
	}{
		{
			name:        "valid date - standard format",
			queryDate:   "2024-01-15",
			expectError: false,
			description: "Should parse YYYY-MM-DD format",
		},
		{
			name:        "valid date - start of year",
			queryDate:   "2024-01-01",
			expectError: false,
			description: "Should parse first day of year",
		},
		{
			name:        "valid date - end of year",
			queryDate:   "2024-12-31",
			expectError: false,
			description: "Should parse last day of year",
		},
		{
			name:        "invalid date - wrong format",
			queryDate:   "01-15-2024",
			expectError: true,
			description: "Should reject MM-DD-YYYY format",
		},
		{
			name:        "invalid date - with slash format",
			queryDate:   "2024/01/15",
			expectError: true,
			description: "Should reject date with slashes",
		},
		{
			name:        "invalid date - not a date",
			queryDate:   "not-a-date",
			expectError: true,
			description: "Should reject non-date strings",
		},
		{
			name:        "invalid date - empty string handled as today",
			queryDate:   "",
			expectError: false,
			description: "Should use today for empty date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockShutdownGetter{
				getShutdownsFunc: func(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
					return []*shutdown.ResponseModel{}, nil
				},
				getOrgTypesMapFunc: func(ctx context.Context) (map[int64][]string, error) {
					return map[int64][]string{}, nil
				},
			}

			url := "/shutdowns"
			if tt.queryDate != "" {
				url += "?date=" + tt.queryDate
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			handler := Get(logger, mock, loc)
			handler.ServeHTTP(rr, req)

			if tt.expectError {
				if rr.Code == http.StatusOK {
					t.Errorf("%s: expected error, but got success", tt.description)
				}
			} else {
				if rr.Code != http.StatusOK {
					t.Errorf("%s: expected success, but got status %v", tt.description, rr.Code)
				}
			}

			t.Logf("%s: PASSED", tt.description)
		})
	}
}
