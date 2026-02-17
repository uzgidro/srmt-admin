package fast_call

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/fast_call"
)

type RepoInterface interface {
	AddFastCall(ctx context.Context, req dto.AddFastCallRequest) (int64, error)
	GetAllFastCalls(ctx context.Context) ([]*fast_call.Model, error)
	GetFastCallByID(ctx context.Context, id int64) (*fast_call.Model, error)
	EditFastCall(ctx context.Context, id int64, req dto.EditFastCallRequest) error
	DeleteFastCall(ctx context.Context, id int64) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddFastCall(ctx context.Context, req dto.AddFastCallRequest) (int64, error) {
	return s.repo.AddFastCall(ctx, req)
}

func (s *Service) GetAllFastCalls(ctx context.Context) ([]*fast_call.Model, error) {
	return s.repo.GetAllFastCalls(ctx)
}

func (s *Service) GetFastCallByID(ctx context.Context, id int64) (*fast_call.Model, error) {
	return s.repo.GetFastCallByID(ctx, id)
}

func (s *Service) EditFastCall(ctx context.Context, id int64, req dto.EditFastCallRequest) error {
	return s.repo.EditFastCall(ctx, id, req)
}

func (s *Service) DeleteFastCall(ctx context.Context, id int64) error {
	return s.repo.DeleteFastCall(ctx, id)
}
