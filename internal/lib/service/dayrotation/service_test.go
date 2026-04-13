package dayrotation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	gesreport "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/weather"
	"srmt-admin/internal/storage/repo"
)

type mockRotator struct {
	result *repo.DayRotationResult
	err    error
}

func (m *mockRotator) RotateDayBoundary(ctx context.Context, cutoff time.Time) (*repo.DayRotationResult, error) {
	return m.result, m.err
}

type mockCascadeGetter struct {
	configs []gesreport.CascadeConfig
	err     error
}

func (m *mockCascadeGetter) GetAllCascadeConfigs(ctx context.Context) ([]gesreport.CascadeConfig, error) {
	return m.configs, m.err
}

type mockWeatherUpdater struct {
	calls []weatherUpdateCall
	err   error
}

type weatherUpdateCall struct {
	OrgID     int64
	Date      string
	Temp      *float64
	Condition *string
}

func (m *mockWeatherUpdater) UpsertCascadeDailyWeather(ctx context.Context, orgID int64, date string, temperature *float64, weatherCondition *string) error {
	m.calls = append(m.calls, weatherUpdateCall{orgID, date, temperature, weatherCondition})
	return m.err
}

type mockWeatherFetcher struct {
	data *weather.WeatherData
	err  error
}

func (m *mockWeatherFetcher) FetchDaily(ctx context.Context, lat, lon float64) (*weather.WeatherData, error) {
	return m.data, m.err
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustLoadLocation(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}
	return loc
}

func newTestService(rotator Rotator, cascades CascadeConfigGetter, weatherRepo WeatherUpdater, fetcher WeatherFetcher, loc *time.Location) *Service {
	return NewService(rotator, cascades, weatherRepo, fetcher, loc, newTestLogger())
}

func TestRun_Success(t *testing.T) {
	loc := mustLoadLocation(t)
	mock := &mockRotator{
		result: &repo.DayRotationResult{
			LinkedDischargesRotated: 2,
			DischargesRotated:       3,
		},
	}
	cascades := &mockCascadeGetter{configs: nil}
	weatherRepo := &mockWeatherUpdater{}
	fetcher := &mockWeatherFetcher{}

	svc := newTestService(mock, cascades, weatherRepo, fetcher, loc)
	svc.Run(context.Background(), time.Now())
}

func TestRun_Error(t *testing.T) {
	loc := mustLoadLocation(t)
	mock := &mockRotator{
		err: errors.New("db error"),
	}
	cascades := &mockCascadeGetter{configs: nil}
	weatherRepo := &mockWeatherUpdater{}
	fetcher := &mockWeatherFetcher{}

	svc := newTestService(mock, cascades, weatherRepo, fetcher, loc)
	svc.Run(context.Background(), time.Now())
}

func TestStartScheduler_ContextCancel(t *testing.T) {
	loc := mustLoadLocation(t)
	mock := &mockRotator{
		result: &repo.DayRotationResult{},
	}
	cascades := &mockCascadeGetter{configs: nil}
	weatherRepo := &mockWeatherUpdater{}
	fetcher := &mockWeatherFetcher{}

	svc := newTestService(mock, cascades, weatherRepo, fetcher, loc)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		svc.StartScheduler(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// goroutine exited as expected
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler goroutine did not exit after context cancellation")
	}
}

func TestFetchWeather_Success(t *testing.T) {
	loc := mustLoadLocation(t)
	lat, lon := 41.3, 69.24

	mock := &mockRotator{
		result: &repo.DayRotationResult{},
	}
	cascades := &mockCascadeGetter{
		configs: []gesreport.CascadeConfig{
			{OrganizationID: 1, OrganizationName: "GES-1", Latitude: &lat, Longitude: &lon},
		},
	}
	weatherRepo := &mockWeatherUpdater{}
	fetcher := &mockWeatherFetcher{
		data: &weather.WeatherData{Temperature: 18.5, Icon: "10d"},
	}

	svc := newTestService(mock, cascades, weatherRepo, fetcher, loc)

	cutoff := time.Date(2026, 4, 9, 5, 0, 0, 0, loc)
	svc.Run(context.Background(), cutoff)

	if len(weatherRepo.calls) != 1 {
		t.Fatalf("expected 1 weather update call, got %d", len(weatherRepo.calls))
	}

	call := weatherRepo.calls[0]
	if call.OrgID != 1 {
		t.Errorf("expected org_id 1, got %d", call.OrgID)
	}
	if call.Date != "2026-04-09" {
		t.Errorf("expected date 2026-04-09, got %s", call.Date)
	}
	if *call.Temp != 18.5 {
		t.Errorf("expected temp 18.5, got %f", *call.Temp)
	}
	if *call.Condition != "10d" {
		t.Errorf("expected condition '10d', got %q", *call.Condition)
	}
}

func TestFetchWeather_SkipMissingCoordinates(t *testing.T) {
	loc := mustLoadLocation(t)

	mock := &mockRotator{result: &repo.DayRotationResult{}}
	cascades := &mockCascadeGetter{
		configs: []gesreport.CascadeConfig{
			{OrganizationID: 1, OrganizationName: "GES-1"}, // no lat/lon
		},
	}
	weatherRepo := &mockWeatherUpdater{}
	fetcher := &mockWeatherFetcher{
		data: &weather.WeatherData{Temperature: 20, Icon: "01d"},
	}

	svc := newTestService(mock, cascades, weatherRepo, fetcher, loc)
	svc.Run(context.Background(), time.Date(2026, 4, 9, 5, 0, 0, 0, loc))

	if len(weatherRepo.calls) != 0 {
		t.Errorf("expected 0 weather update calls for cascade without coordinates, got %d", len(weatherRepo.calls))
	}
}

func TestFetchWeather_ContinuesOnError(t *testing.T) {
	loc := mustLoadLocation(t)
	lat1, lon1 := 41.3, 69.24
	lat2, lon2 := 40.5, 68.8

	mock := &mockRotator{result: &repo.DayRotationResult{}}
	cascades := &mockCascadeGetter{
		configs: []gesreport.CascadeConfig{
			{OrganizationID: 1, Latitude: &lat1, Longitude: &lon1},
			{OrganizationID: 2, Latitude: &lat2, Longitude: &lon2},
		},
	}
	weatherRepo := &mockWeatherUpdater{}
	// fetcher returns error — should continue to next cascade
	fetcher := &mockWeatherFetcher{
		err: errors.New("API timeout"),
	}

	svc := newTestService(mock, cascades, weatherRepo, fetcher, loc)
	svc.Run(context.Background(), time.Date(2026, 4, 9, 5, 0, 0, 0, loc))

	// No updates should happen since all fetches failed
	if len(weatherRepo.calls) != 0 {
		t.Errorf("expected 0 weather update calls on fetch error, got %d", len(weatherRepo.calls))
	}
}
