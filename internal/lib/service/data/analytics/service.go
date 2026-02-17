package analytics

import (
	"context"
	"log/slog"
	complexValue "srmt-admin/internal/lib/model/dto/complex-value"
)

type RepoInterface interface {
	GetSelectedYearDataIncome(ctx context.Context, id, year int) (complexValue.Model, error)
	GetDataByYears(ctx context.Context, id int) (complexValue.Model, error)
	GetAvgData(ctx context.Context, id int) (complexValue.Model, error)
	GetTenYearsAvgData(ctx context.Context, id int) (complexValue.Model, error)
	GetExtremumYear(ctx context.Context, id int, extremumType string) (int, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetSelectedYearDataIncome(ctx context.Context, id, year int) (complexValue.Model, error) {
	return s.repo.GetSelectedYearDataIncome(ctx, id, year)
}

func (s *Service) GetDataByYears(ctx context.Context, id int) (complexValue.Model, error) {
	return s.repo.GetDataByYears(ctx, id)
}

func (s *Service) GetAvgData(ctx context.Context, id int) (complexValue.Model, error) {
	return s.repo.GetAvgData(ctx, id)
}

func (s *Service) GetTenYearsAvgData(ctx context.Context, id int) (complexValue.Model, error) {
	return s.repo.GetTenYearsAvgData(ctx, id)
}

func (s *Service) GetExtremumYear(ctx context.Context, id int, extremumType string) (int, error) {
	return s.repo.GetExtremumYear(ctx, id, extremumType)
}
