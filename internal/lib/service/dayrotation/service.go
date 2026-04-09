package dayrotation

import (
	"context"
	"log/slog"
	"srmt-admin/internal/storage/repo"
	"time"
)

type Rotator interface {
	RotateDayBoundary(ctx context.Context, cutoff time.Time) (*repo.DayRotationResult, error)
}

type Service struct {
	log        *slog.Logger
	repo       Rotator
	loc        *time.Location
	runHour    int // when to actually run (schedule)
	cutoffHour int // what time to record as the day boundary
}

func NewService(repo Rotator, loc *time.Location, log *slog.Logger) *Service {
	return &Service{
		log:        log.With(slog.String("service", "dayrotation")),
		repo:       repo,
		loc:        loc,
		runHour:    4, // run at 04:00 Tashkent
		cutoffHour: 5, // record cutoff as 05:00
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
		slog.Int("infra_events_rotated", result.InfraEventsRotated),
	)
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
