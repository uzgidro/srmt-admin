package position

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/position"
)

type RepoInterface interface {
	AddPosition(ctx context.Context, name string, description *string) (int64, error)
	GetAllPositions(ctx context.Context) ([]*position.Model, error)
	EditPosition(ctx context.Context, id int64, name *string, description *string) error
	DeletePosition(ctx context.Context, id int64) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddPosition(ctx context.Context, req dto.AddPositionRequest) (int64, error) {
	return s.repo.AddPosition(ctx, req.Name, req.Description)
}

func (s *Service) GetAllPositions(ctx context.Context) ([]*position.Model, error) {
	return s.repo.GetAllPositions(ctx)
}

func (s *Service) EditPosition(ctx context.Context, id int64, req dto.EditPositionRequest) error {
	return s.repo.EditPosition(ctx, id, req.Name, req.Description)
}

func (s *Service) DeletePosition(ctx context.Context, id int64) error {
	return s.repo.DeletePosition(ctx, id)
}
