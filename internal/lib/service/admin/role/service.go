package role

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/model/role"
	"srmt-admin/internal/lib/model/user"
)

type RepoInterface interface {
	AddRole(ctx context.Context, name string, description string) (int64, error)
	GetAllRoles(ctx context.Context) ([]role.Model, error)
	EditRole(ctx context.Context, id int64, name, description string) error
	DeleteRole(ctx context.Context, id int64) error
	GetUsersByRole(ctx context.Context, roleID int64) ([]user.Model, error)
	AssignRoleToUsers(ctx context.Context, roleID int64, userIDs []int64) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddRole(ctx context.Context, name string, description string) (int64, error) {
	return s.repo.AddRole(ctx, name, description)
}

func (s *Service) GetAllRoles(ctx context.Context) ([]role.Model, error) {
	return s.repo.GetAllRoles(ctx)
}

func (s *Service) EditRole(ctx context.Context, id int64, name, description string) error {
	return s.repo.EditRole(ctx, id, name, description)
}

func (s *Service) DeleteRole(ctx context.Context, id int64) error {
	return s.repo.DeleteRole(ctx, id)
}

func (s *Service) GetUsersByRole(ctx context.Context, roleID int64) ([]user.Model, error) {
	return s.repo.GetUsersByRole(ctx, roleID)
}

func (s *Service) AssignRoleToUsers(ctx context.Context, roleID int64, userIDs []int64) error {
	return s.repo.AssignRoleToUsers(ctx, roleID, userIDs)
}
