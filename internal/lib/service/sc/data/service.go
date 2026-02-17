package data

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	gesproduction "srmt-admin/internal/lib/model/ges-production"
	"srmt-admin/internal/lib/model/levelvolume"
	reservoirdata "srmt-admin/internal/lib/model/reservoir-data"
	reservoirdevicesummary "srmt-admin/internal/lib/model/reservoir-device-summary"
	reservoirsummary "srmt-admin/internal/lib/model/reservoir-summary"
	"srmt-admin/internal/storage/repo"
)

type RepoInterface interface {
	// Indicators
	SetIndicator(ctx context.Context, resID int64, height float64) (int64, error)

	// Level-volume
	GetLevelVolume(ctx context.Context, organizationID int64, level float64) (*levelvolume.Model, error)

	// Reservoir device summary
	GetReservoirDeviceSummary(ctx context.Context) ([]*reservoirdevicesummary.ResponseModel, error)
	PatchReservoirDeviceSummary(ctx context.Context, req dto.PatchReservoirDeviceSummaryRequest, updatedByUserID int64) error

	// Reservoir summary
	GetReservoirSummary(ctx context.Context, date string) ([]*reservoirsummary.ResponseModel, error)
	UpsertReservoirData(ctx context.Context, data []reservoirdata.ReservoirDataItem, userID int64) error

	// Snow cover
	GetSnowCoverByDates(ctx context.Context, dates []string) ([]repo.SnowCoverRow, error)
	UpsertSnowCoverBatch(ctx context.Context, date string, resourceDate string, items []repo.SnowCoverItem) error

	// GES production
	UpsertGesProduction(ctx context.Context, data gesproduction.Model) error

	// Reservoirs
	AddReservoir(ctx context.Context, name string) (int64, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) SetIndicator(ctx context.Context, resID int64, height float64) (int64, error) {
	return s.repo.SetIndicator(ctx, resID, height)
}

func (s *Service) GetLevelVolume(ctx context.Context, organizationID int64, level float64) (*levelvolume.Model, error) {
	return s.repo.GetLevelVolume(ctx, organizationID, level)
}

func (s *Service) GetReservoirDeviceSummary(ctx context.Context) ([]*reservoirdevicesummary.ResponseModel, error) {
	return s.repo.GetReservoirDeviceSummary(ctx)
}

func (s *Service) PatchReservoirDeviceSummary(ctx context.Context, req dto.PatchReservoirDeviceSummaryRequest, updatedByUserID int64) error {
	return s.repo.PatchReservoirDeviceSummary(ctx, req, updatedByUserID)
}

func (s *Service) GetReservoirSummary(ctx context.Context, date string) ([]*reservoirsummary.ResponseModel, error) {
	return s.repo.GetReservoirSummary(ctx, date)
}

func (s *Service) UpsertReservoirData(ctx context.Context, data []reservoirdata.ReservoirDataItem, userID int64) error {
	return s.repo.UpsertReservoirData(ctx, data, userID)
}

func (s *Service) GetSnowCoverByDates(ctx context.Context, dates []string) ([]repo.SnowCoverRow, error) {
	return s.repo.GetSnowCoverByDates(ctx, dates)
}

func (s *Service) UpsertSnowCoverBatch(ctx context.Context, date string, resourceDate string, items []repo.SnowCoverItem) error {
	return s.repo.UpsertSnowCoverBatch(ctx, date, resourceDate, items)
}

func (s *Service) UpsertGesProduction(ctx context.Context, data gesproduction.Model) error {
	return s.repo.UpsertGesProduction(ctx, data)
}

func (s *Service) AddReservoir(ctx context.Context, name string) (int64, error) {
	return s.repo.AddReservoir(ctx, name)
}
