package pastevents

import (
	"context"
	"log/slog"
	past_events "srmt-admin/internal/lib/dto/past-events"
	"time"
)

type RepoInterface interface {
	GetPastEvents(ctx context.Context, days int, timezone *time.Location) ([]past_events.DateGroup, error)
	GetPastEventsByDate(ctx context.Context, date time.Time, timezone *time.Location) ([]past_events.DateGroup, error)
	GetPastEventsByDateAndType(ctx context.Context, date time.Time, eventType string, timezone *time.Location) ([]past_events.Event, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetPastEvents(ctx context.Context, days int, timezone *time.Location) ([]past_events.DateGroup, error) {
	return s.repo.GetPastEvents(ctx, days, timezone)
}

func (s *Service) GetPastEventsByDate(ctx context.Context, date time.Time, timezone *time.Location) ([]past_events.DateGroup, error) {
	return s.repo.GetPastEventsByDate(ctx, date, timezone)
}

func (s *Service) GetPastEventsByDateAndType(ctx context.Context, date time.Time, eventType string, timezone *time.Location) ([]past_events.Event, error) {
	return s.repo.GetPastEventsByDateAndType(ctx, date, eventType, timezone)
}
