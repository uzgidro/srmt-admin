package reservoirsummary

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	reservoirdata "srmt-admin/internal/lib/model/reservoir-data"
	optional "srmt-admin/internal/lib/optional"
	"srmt-admin/internal/token"
)

type captureUpserter struct {
	mu   sync.Mutex
	last []reservoirdata.ReservoirDataItem
}

func (c *captureUpserter) UpsertReservoirData(_ context.Context, data []reservoirdata.ReservoirDataItem, _ int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = data
	return nil
}

func TestUpsertForwardsOptionalFieldStates(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		check func(t *testing.T, item reservoirdata.ReservoirDataItem)
	}{
		{
			name: "all four numeric fields absent",
			body: `[{"organization_id":1,"date":"2026-04-12"}]`,
			check: func(t *testing.T, item reservoirdata.ReservoirDataItem) {
				for name, got := range map[string]optional.Optional[float64]{
					"income":  item.Income,
					"level":   item.Level,
					"release": item.Release,
					"volume":  item.Volume,
				} {
					if got.Set {
						t.Errorf("%s: Set=true, want false", name)
					}
					if got.Value != nil {
						t.Errorf("%s: Value=%v, want nil", name, *got.Value)
					}
				}
			},
		},
		{
			name: "all four explicit null",
			body: `[{"organization_id":1,"date":"2026-04-12","income":null,"level":null,"release":null,"volume":null}]`,
			check: func(t *testing.T, item reservoirdata.ReservoirDataItem) {
				for name, got := range map[string]optional.Optional[float64]{
					"income":  item.Income,
					"level":   item.Level,
					"release": item.Release,
					"volume":  item.Volume,
				} {
					if !got.Set {
						t.Errorf("%s: Set=false, want true", name)
					}
					if got.Value != nil {
						t.Errorf("%s: Value=%v, want nil", name, *got.Value)
					}
				}
			},
		},
		{
			name: "all four numeric",
			body: `[{"organization_id":1,"date":"2026-04-12","income":1,"level":2,"release":3,"volume":4}]`,
			check: func(t *testing.T, item reservoirdata.ReservoirDataItem) {
				pairs := []struct {
					name string
					got  optional.Optional[float64]
					want float64
				}{
					{"income", item.Income, 1},
					{"level", item.Level, 2},
					{"release", item.Release, 3},
					{"volume", item.Volume, 4},
				}
				for _, p := range pairs {
					if !p.got.Set {
						t.Errorf("%s: Set=false, want true", p.name)
						continue
					}
					if p.got.Value == nil {
						t.Errorf("%s: Value=nil, want %v", p.name, p.want)
						continue
					}
					if *p.got.Value != p.want {
						t.Errorf("%s: Value=%v, want %v", p.name, *p.got.Value, p.want)
					}
				}
			},
		},
		{
			name: "mixed: income number, level null, release absent, volume 0",
			body: `[{"organization_id":1,"date":"2026-04-12","income":7.5,"level":null,"volume":0}]`,
			check: func(t *testing.T, item reservoirdata.ReservoirDataItem) {
				if !item.Income.Set || item.Income.Value == nil || *item.Income.Value != 7.5 {
					t.Errorf("income: want Set=true Value=7.5, got Set=%v Value=%v", item.Income.Set, item.Income.Value)
				}
				if !item.Level.Set || item.Level.Value != nil {
					t.Errorf("level: want Set=true Value=nil, got Set=%v Value=%v", item.Level.Set, item.Level.Value)
				}
				if item.Release.Set {
					t.Errorf("release: want Set=false, got Set=%v Value=%v", item.Release.Set, item.Release.Value)
				}
				if !item.Volume.Set || item.Volume.Value == nil || *item.Volume.Value != 0 {
					t.Errorf("volume: want Set=true Value=0, got Set=%v Value=%v", item.Volume.Set, item.Volume.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := slog.New(slog.NewTextHandler(os.Stdout, nil))
			cap := &captureUpserter{}
			handler := New(log, cap)

			verifier := &mockTokenVerifier{claims: &token.Claims{
				UserID:         1,
				OrganizationID: 1,
				Roles:          []string{"sc"},
			}}

			r := chi.NewRouter()
			r.Use(mwauth.Authenticator(verifier))
			r.Post("/reservoir-summary", handler)

			req := httptest.NewRequest(http.MethodPost, "/reservoir-summary", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			if len(cap.last) != 1 {
				t.Fatalf("captured %d items, want 1", len(cap.last))
			}
			tt.check(t, cap.last[0])
		})
	}
}
