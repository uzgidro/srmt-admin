package department

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/department"
)

type RepoInterface interface {
	AddDepartment(ctx context.Context, name string, description *string, orgID int64) (int64, error)
	GetAllDepartments(ctx context.Context, orgID *int64) ([]*department.Model, error)
	GetDepartmentByID(ctx context.Context, id int64) (*department.Model, error)
	EditDepartment(ctx context.Context, id int64, name *string, description *string) error
	DeleteDepartment(ctx context.Context, id int64) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddDepartment(ctx context.Context, req dto.AddDepartmentRequest) (int64, error) {
	return s.repo.AddDepartment(ctx, req.Name, req.Description, req.OrganizationID)
}

func (s *Service) GetAllDepartments(ctx context.Context, orgID *int64) ([]*department.Model, error) {
	return s.repo.GetAllDepartments(ctx, orgID)
}

func (s *Service) GetDepartmentByID(ctx context.Context, id int64) (*department.Model, error) {
	return s.repo.GetDepartmentByID(ctx, id)
}

func (s *Service) EditDepartment(ctx context.Context, id int64, req dto.EditDepartmentRequest) error {
	return s.repo.EditDepartment(ctx, id, req.Name, req.Description)
}

func (s *Service) DeleteDepartment(ctx context.Context, id int64) error {
	return s.repo.DeleteDepartment(ctx, id)
}
