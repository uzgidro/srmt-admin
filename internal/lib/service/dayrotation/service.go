package dayrotation

import (
	"context"
	"log/slog"
	gesreport "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/weather"
	"srmt-admin/internal/storage/repo"
	"time"
)

type Rotator interface {
	RotateDayBoundary(ctx context.Context, cutoff time.Time) (*repo.DayRotationResult, error)
}

type CascadeConfigGetter interface {
	GetAllCascadeConfigs(ctx context.Context) ([]gesreport.CascadeConfig, error)
}

type WeatherUpdater interface {
	UpsertWeatherData(ctx context.Context, orgID int64, date string, temp *float64, condition *string) error
}

type WeatherFetcher interface {
	FetchDaily(ctx context.Context, lat, lon float64) (*weather.WeatherData, error)
}

type Service struct {
	log            *slog.Logger
	repo           Rotator
	cascades       CascadeConfigGetter
	weatherRepo    WeatherUpdater
	weatherFetcher WeatherFetcher
	loc            *time.Location
	runHour        int // when to actually run (schedule)
	cutoffHour     int // what time to record as the day boundary
}

func NewService(
	repo Rotator,
	cascades CascadeConfigGetter,
	weatherRepo WeatherUpdater,
	weatherFetcher WeatherFetcher,
	loc *time.Location,
	log *slog.Logger,
) *Service {
	return &Service{
		log:            log.With(slog.String("service", "dayrotation")),
		repo:           repo,
		cascades:       cascades,
		weatherRepo:    weatherRepo,
		weatherFetcher: weatherFetcher,
		loc:            loc,
		runHour:        4, // run at 04:00 Tashkent
		cutoffHour:     5, // record cutoff as 05:00
	}
}

// Run performs one rotation cycle at the given cutoff time.
func (s *Service) Run(ctx context.Context, cutoff time.Time) {
	s.log.Info("starting day rotation", slog.String("cutoff", cutoff.Format(time.RFC3339)))

	result, err := s.repo.RotateDayBoundary(ctx, cutoff)
	if err != nil {
		s.log.Error("day rotation failed", slog.String("error", err.Error()))
		return
	}

	s.log.Info("day rotation completed",
		slog.Int("linked_discharges_rotated", result.LinkedDischargesRotated),
		slog.Int("discharges_rotated", result.DischargesRotated),
	)

	// Fetch weather data for each cascade
	date := cutoff.In(s.loc).Format("2006-01-02")
	s.fetchWeather(ctx, date)
}

// fetchWeather fetches weather data for all cascades and stores it in ges_daily_data.
func (s *Service) fetchWeather(ctx context.Context, date string) {
	if s.weatherFetcher == nil {
		s.log.Warn("weather fetcher not configured, skipping weather fetch")
		return
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	configs, err := s.cascades.GetAllCascadeConfigs(fetchCtx)
	if err != nil {
		s.log.Error("failed to get cascade configs for weather", slog.String("error", err.Error()))
		return
	}

	var fetched, failed int
	for _, cfg := range configs {
		if cfg.Latitude == nil || cfg.Longitude == nil {
			s.log.Warn("cascade has no coordinates, skipping weather",
				slog.Int64("organization_id", cfg.OrganizationID),
				slog.String("organization", cfg.OrganizationName))
			continue
		}

		data, err := s.weatherFetcher.FetchDaily(fetchCtx, *cfg.Latitude, *cfg.Longitude)
		if err != nil {
			s.log.Error("failed to fetch weather",
				slog.Int64("organization_id", cfg.OrganizationID),
				slog.String("organization", cfg.OrganizationName),
				slog.String("error", err.Error()))
			failed++
			continue
		}

		if err := s.weatherRepo.UpsertWeatherData(fetchCtx, cfg.OrganizationID, date, &data.Temperature, &data.Icon); err != nil {
			s.log.Error("failed to save weather data",
				slog.Int64("organization_id", cfg.OrganizationID),
				slog.String("error", err.Error()))
			failed++
			continue
		}

		fetched++
	}

	s.log.Info("weather fetch completed",
		slog.String("date", date),
		slog.Int("fetched", fetched),
		slog.Int("failed", failed),
		slog.Int("total_cascades", len(configs)))
}

// StartScheduler runs the rotation daily at the configured hour. Blocks until ctx is cancelled.
func (s *Service) StartScheduler(ctx context.Context) {
	for {
		now := time.Now().In(s.loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), s.runHour, 0, 0, 0, s.loc)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		wait := next.Sub(now)

		cutoff := time.Date(next.Year(), next.Month(), next.Day(), s.cutoffHour, 0, 0, 0, s.loc)

		s.log.Info("next day rotation scheduled",
			slog.String("run_at", next.Format(time.RFC3339)),
			slog.String("cutoff", cutoff.Format(time.RFC3339)),
			slog.Duration("in", wait))

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.log.Info("day rotation scheduler stopped")
			return
		case <-timer.C:
			s.Run(ctx, cutoff)
		}
	}
}
