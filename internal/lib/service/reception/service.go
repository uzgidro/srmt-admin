package reception

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/reception"
)

type RepoInterface interface {
	AddReception(ctx context.Context, req dto.AddReceptionRequest) (int64, error)
	GetAllReceptions(ctx context.Context, filters dto.GetAllReceptionsFilters) ([]*reception.Model, error)
	GetReceptionByID(ctx context.Context, id int64) (*reception.Model, error)
	EditReception(ctx context.Context, receptionID int64, req dto.EditReceptionRequest) error
	DeleteReception(ctx context.Context, id int64) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddReception(ctx context.Context, req dto.AddReceptionRequest) (int64, error) {
	return s.repo.AddReception(ctx, req)
}

func (s *Service) GetAllReceptions(ctx context.Context, filters dto.GetAllReceptionsFilters) ([]*reception.Model, error) {
	return s.repo.GetAllReceptions(ctx, filters)
}

func (s *Service) GetReceptionByID(ctx context.Context, id int64) (*reception.Model, error) {
	return s.repo.GetReceptionByID(ctx, id)
}

func (s *Service) EditReception(ctx context.Context, receptionID int64, req dto.EditReceptionRequest) error {
	return s.repo.EditReception(ctx, receptionID, req)
}

func (s *Service) DeleteReception(ctx context.Context, id int64) error {
	return s.repo.DeleteReception(ctx, id)
}
