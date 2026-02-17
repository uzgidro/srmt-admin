package telemetry

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/model/asutp"
)

type RepoInterface interface {
	SaveTelemetry(ctx context.Context, stationDBID int64, env *asutp.Envelope) error
	GetStationTelemetry(ctx context.Context, stationDBID int64) ([]*asutp.Envelope, error)
	GetDeviceTelemetry(ctx context.Context, stationDBID int64, deviceID string) (*asutp.Envelope, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) SaveTelemetry(ctx context.Context, stationDBID int64, env *asutp.Envelope) error {
	return s.repo.SaveTelemetry(ctx, stationDBID, env)
}

func (s *Service) GetStationTelemetry(ctx context.Context, stationDBID int64) ([]*asutp.Envelope, error) {
	return s.repo.GetStationTelemetry(ctx, stationDBID)
}

func (s *Service) GetDeviceTelemetry(ctx context.Context, stationDBID int64, deviceID string) (*asutp.Envelope, error) {
	return s.repo.GetDeviceTelemetry(ctx, stationDBID, deviceID)
}
