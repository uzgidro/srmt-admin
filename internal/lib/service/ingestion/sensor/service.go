package sensor

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/model/data"
)

type RepoInterface interface {
	GetIndicator(ctx context.Context, resID int64) (float64, error)
	GetVolumeByLevel(ctx context.Context, resID int64, level float64) (float64, error)
	SaveData(ctx context.Context, data data.Model) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetIndicator(ctx context.Context, resID int64) (float64, error) {
	return s.repo.GetIndicator(ctx, resID)
}

func (s *Service) GetVolumeByLevel(ctx context.Context, resID int64, level float64) (float64, error) {
	return s.repo.GetVolumeByLevel(ctx, resID, level)
}

func (s *Service) SaveData(ctx context.Context, data data.Model) error {
	return s.repo.SaveData(ctx, data)
}
