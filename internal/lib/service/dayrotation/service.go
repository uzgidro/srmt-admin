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
	log  *slog.Logger
	repo Rotator
	loc  *time.Location
	hour int
}

func NewService(repo Rotator, loc *time.Location, log *slog.Logger) *Service {
	return &Service{
		log:  log.With(slog.String("service", "dayrotation")),
		repo: repo,
		loc:  loc,
		hour: 5, // 05:00 Tashkent
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
}

// StartScheduler runs the rotation daily at the configured hour. Blocks until ctx is cancelled.
func (s *Service) StartScheduler(ctx context.Context) {
	for {
		now := time.Now().In(s.loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), s.hour, 0, 0, 0, s.loc)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		wait := next.Sub(now)

		s.log.Info("next day rotation scheduled", slog.String("at", next.Format(time.RFC3339)), slog.Duration("in", wait))

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.log.Info("day rotation scheduler stopped")
			return
		case <-timer.C:
			s.Run(ctx, next)
		}
	}
}
