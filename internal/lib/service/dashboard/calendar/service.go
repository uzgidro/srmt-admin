package calendar

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"time"
)

type RepoInterface interface {
	GetCalendarEventsCounts(ctx context.Context, year, month int, timezone *time.Location) (map[string]*dto.DayCounts, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetCalendarEventsCounts(ctx context.Context, year, month int, timezone *time.Location) (map[string]*dto.DayCounts, error) {
	return s.repo.GetCalendarEventsCounts(ctx, year, month, timezone)
}
