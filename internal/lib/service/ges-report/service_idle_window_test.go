package gesreportservice

import (
	"context"
	"testing"
	"time"

	model "srmt-admin/internal/lib/model/ges-report"
)

// idleWindowMock is a thin Repository implementation that captures the
// (start, end) window passed to GetIdleDischargesForDate and emulates the
// SQL clipping the new repo will perform: returned rows carry only the
// portion of each discharge that overlaps with the window.
type idleWindowMock struct {
	mockRepo
	gotStart time.Time
	gotEnd   time.Time
	rawSpans []idleSpan // input spans; clipped on the fly
}

// idleSpan describes a discharge as it sits in the DB. The mock clips it to
// the request window and returns one IdleDischargeRow with the clipped
// volume — exactly what the production SQL is expected to do.
type idleSpan struct {
	orgID     int64
	flowM3s   float64
	start     time.Time
	end       *time.Time // nil = ongoing
	reason    string
	isOngoing bool
}

func (m *idleWindowMock) GetIdleDischargesForDate(_ context.Context, start, end time.Time) ([]model.IdleDischargeRow, error) {
	m.gotStart = start
	m.gotEnd = end

	out := make([]model.IdleDischargeRow, 0, len(m.rawSpans))
	for _, s := range m.rawSpans {
		// Production WHERE: start < end AND (end > start OR end IS NULL).
		if !s.start.Before(end) {
			continue
		}
		spanEnd := end
		if s.end != nil {
			if !s.end.After(start) {
				continue
			}
			if s.end.Before(spanEnd) {
				spanEnd = *s.end
			}
		}
		spanStart := start
		if s.start.After(spanStart) {
			spanStart = s.start
		}
		seconds := spanEnd.Sub(spanStart).Seconds()
		if seconds <= 0 {
			continue
		}
		volMln := seconds * s.flowM3s / 1_000_000
		reason := s.reason
		out = append(out, model.IdleDischargeRow{
			OrganizationID: s.orgID,
			FlowRateM3s:    s.flowM3s,
			VolumeMlnM3:    volMln,
			Reason:         &reason,
			IsOngoing:      s.isOngoing,
		})
	}
	return out, nil
}

// TestBuildDailyReport_IdleClipsToCalendarDayWindow pins three things in one
// table-driven test:
//   1. The service requests idle discharges over a calendar-day window
//      (00:00 → 24:00 in Asia/Tashkent), not the legacy 05:00 operational
//      window. If the service ever sends 05:00 again, the dayStart assertion
//      below catches it.
//   2. A discharge that starts before midnight contributes only the portion
//      that overlaps with the window — not its full duration.
//   3. A discharge that ends after midnight contributes only the portion
//      inside the window.
//
// The mock emulates the SQL the production repo will run, so this test
// pins the wire contract end-to-end as the report sees it. The volume → flow
// conversion (volume / 0.0864) lives in buildDischargeMap and is exercised
// implicitly by checking IdleDischarge.FlowRateM3s.
func TestBuildDailyReport_IdleClipsToCalendarDayWindow(t *testing.T) {
	loc := mustLoc("Asia/Tashkent")
	const orgID int64 = 100
	const flow = 10.0

	// Window: 22.04 00:00 → 23.04 00:00 (24h = 86400 s).
	wantStart := time.Date(2026, 4, 22, 0, 0, 0, 0, loc)
	wantEnd := wantStart.Add(24 * time.Hour)

	at := func(y int, m time.Month, d, hh int) time.Time {
		return time.Date(y, m, d, hh, 0, 0, 0, loc)
	}
	pt := func(t time.Time) *time.Time { return &t }

	cascadeID := int64(1)
	cascadeName := "C"
	baseToday := []model.RawDailyRow{{
		OrganizationID:        orgID,
		OrganizationName:      "S",
		CascadeID:             &cascadeID,
		CascadeName:           &cascadeName,
		Date:                  "2026-04-22",
		DailyProductionMlnKWh: 24.0,
		WorkingAggregates:     1,
		InstalledCapacityMWt:  100,
		TotalAggregates:       2,
	}}

	// Expected volume helper: hours_in_window * 3600 * flow / 1M.
	mln := func(hours float64) float64 {
		return hours * 3600 * flow / 1_000_000
	}

	cases := []struct {
		name      string
		span      idleSpan
		wantHours float64
	}{
		{
			name: "starts_before_midnight",
			// 21.04 18:00 → 22.04 06:00 (12h total). Window 22.04 → keep 6h.
			span: idleSpan{
				orgID: orgID, flowM3s: flow,
				start: at(2026, 4, 21, 18), end: pt(at(2026, 4, 22, 6)),
				reason: "yesterday-spillover",
			},
			wantHours: 6,
		},
		{
			name: "ends_after_midnight",
			// 22.04 22:00 → 23.04 04:00 (6h total). Window 22.04 → keep 2h.
			span: idleSpan{
				orgID: orgID, flowM3s: flow,
				start: at(2026, 4, 22, 22), end: pt(at(2026, 4, 23, 4)),
				reason: "tomorrow-spillover",
			},
			wantHours: 2,
		},
		{
			name: "fully_within",
			// 22.04 10:00 → 14:00 (4h). Whole span in window.
			span: idleSpan{
				orgID: orgID, flowM3s: flow,
				start: at(2026, 4, 22, 10), end: pt(at(2026, 4, 22, 14)),
				reason: "midday",
			},
			wantHours: 4,
		},
		{
			// Pins the COALESCE(end_time, NOW()) → window_end branch in SQL
			// (the production query treats NULL end_time as window_end via
			// LEAST). Without this case a future regression that breaks the
			// COALESCE would still pass the other three subcases.
			name: "ongoing_clips_to_window_end",
			// 22.04 20:00 → still running. Window end = 23.04 00:00 → keep 4h.
			span: idleSpan{
				orgID: orgID, flowM3s: flow,
				start: at(2026, 4, 22, 20), end: nil,
				reason:    "ongoing",
				isOngoing: true,
			},
			wantHours: 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &idleWindowMock{
				mockRepo: mockRepo{
					todayDate:     "2026-04-22",
					yesterdayDate: "2026-04-21",
					prevYearDate:  "2025-04-22",
					todayData:     baseToday,
				},
				rawSpans: []idleSpan{tc.span},
			}
			svc := NewService(m, loc, discardLogger())

			report, err := svc.BuildDailyReport(context.Background(), "2026-04-22", nil)
			if err != nil {
				t.Fatalf("BuildDailyReport: %v", err)
			}

			// (1) Window assertion — calendar day, not 05:00.
			if !m.gotStart.Equal(wantStart) {
				t.Errorf("dayStart: want %s, got %s", wantStart, m.gotStart)
			}
			if !m.gotEnd.Equal(wantEnd) {
				t.Errorf("dayEnd: want %s, got %s", wantEnd, m.gotEnd)
			}

			// (2)/(3) Clipped volume → flow.
			st := report.Cascades[0].Stations[0]
			if st.IdleDischarge == nil {
				t.Fatal("IdleDischarge is nil")
			}
			wantVol := mln(tc.wantHours)
			if !approxEqual(st.IdleDischarge.VolumeMlnM3, roundTo2(wantVol)) {
				t.Errorf("VolumeMlnM3: want %.4f (rounded %.2f), got %.4f",
					wantVol, roundTo2(wantVol), st.IdleDischarge.VolumeMlnM3)
			}
			// FlowRate = clipped_volume / 0.0864 (см. buildDischargeMap).
			wantFlow := roundTo2(wantVol / 0.0864)
			if !approxEqual(st.IdleDischarge.FlowRateM3s, wantFlow) {
				t.Errorf("FlowRateM3s: want %.4f, got %.4f", wantFlow, st.IdleDischarge.FlowRateM3s)
			}
			// IsOngoing must propagate through buildDischargeMap.
			if st.IdleDischarge.IsOngoing != tc.span.isOngoing {
				t.Errorf("IsOngoing: want %v, got %v", tc.span.isOngoing, st.IdleDischarge.IsOngoing)
			}
		})
	}
}
