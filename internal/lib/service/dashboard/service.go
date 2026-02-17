package dashboard

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	gesproduction "srmt-admin/internal/lib/model/ges-production"
)

type RepoInterface interface {
	GetCascadesWithDetails(ctx context.Context, ascueFetcher dto.ASCUEFetcher) ([]*dto.CascadeWithDetails, error)
	GetOrganizationsWithReservoir(ctx context.Context, orgIDs []int64, reservoirFetcher dto.ReservoirFetcher, date string) ([]*dto.OrganizationWithReservoir, error)
	GetGesProductionDashboard(ctx context.Context) (*gesproduction.DashboardResponse, error)
	GetGesProductionStats(ctx context.Context) (*gesproduction.StatsResponse, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetCascadesWithDetails(ctx context.Context, ascueFetcher dto.ASCUEFetcher) ([]*dto.CascadeWithDetails, error) {
	return s.repo.GetCascadesWithDetails(ctx, ascueFetcher)
}

func (s *Service) GetOrganizationsWithReservoir(ctx context.Context, orgIDs []int64, reservoirFetcher dto.ReservoirFetcher, date string) ([]*dto.OrganizationWithReservoir, error) {
	return s.repo.GetOrganizationsWithReservoir(ctx, orgIDs, reservoirFetcher, date)
}

func (s *Service) GetGesProductionDashboard(ctx context.Context) (*gesproduction.DashboardResponse, error) {
	return s.repo.GetGesProductionDashboard(ctx)
}

func (s *Service) GetGesProductionStats(ctx context.Context) (*gesproduction.StatsResponse, error) {
	return s.repo.GetGesProductionStats(ctx)
}
