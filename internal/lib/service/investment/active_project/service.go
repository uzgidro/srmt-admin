package active_project

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	investActiveProject "srmt-admin/internal/lib/model/invest-active-project"
)

type RepoInterface interface {
	AddInvestActiveProject(ctx context.Context, req dto.AddInvestActiveProjectRequest) (int64, error)
	GetAllInvestActiveProjects(ctx context.Context) ([]*investActiveProject.Model, error)
	GetInvestActiveProjectByID(ctx context.Context, id int64) (*investActiveProject.Model, error)
	EditInvestActiveProject(ctx context.Context, id int64, req dto.EditInvestActiveProjectRequest) error
	DeleteInvestActiveProject(ctx context.Context, id int64) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddInvestActiveProject(ctx context.Context, req dto.AddInvestActiveProjectRequest) (int64, error) {
	return s.repo.AddInvestActiveProject(ctx, req)
}

func (s *Service) GetAllInvestActiveProjects(ctx context.Context) ([]*investActiveProject.Model, error) {
	return s.repo.GetAllInvestActiveProjects(ctx)
}

func (s *Service) GetInvestActiveProjectByID(ctx context.Context, id int64) (*investActiveProject.Model, error) {
	return s.repo.GetInvestActiveProjectByID(ctx, id)
}

func (s *Service) EditInvestActiveProject(ctx context.Context, id int64, req dto.EditInvestActiveProjectRequest) error {
	return s.repo.EditInvestActiveProject(ctx, id, req)
}

func (s *Service) DeleteInvestActiveProject(ctx context.Context, id int64) error {
	return s.repo.DeleteInvestActiveProject(ctx, id)
}
