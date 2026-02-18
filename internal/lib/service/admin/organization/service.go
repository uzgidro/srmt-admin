package organization

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/organization"
	organization_type "srmt-admin/internal/lib/model/organization-type"
)

type RepoInterface interface {
	// Organizations
	GetAllOrganizations(ctx context.Context, orgType *string) ([]*organization.Model, error)
	GetFlatOrganizations(ctx context.Context, orgType *string) ([]*organization.Model, error)
	AddOrganization(ctx context.Context, name string, parentID *int64, typeIDs []int64) (int64, error)
	EditOrganization(ctx context.Context, id int64, name *string, parentID **int64, typeIDs []int64) error
	DeleteOrganization(ctx context.Context, id int64) error

	// Organization Types
	AddOrganizationType(ctx context.Context, name string, description *string) (int64, error)
	GetAllOrganizationTypes(ctx context.Context) ([]organization_type.Model, error)
	EditOrganizationType(ctx context.Context, id int64, name, description *string) error
	DeleteOrganizationType(ctx context.Context, id string) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// Organizations

func (s *Service) GetAllOrganizations(ctx context.Context, orgType *string) ([]*organization.Model, error) {
	return s.repo.GetAllOrganizations(ctx, orgType)
}

func (s *Service) GetFlatOrganizations(ctx context.Context, orgType *string) ([]*organization.Model, error) {
	return s.repo.GetFlatOrganizations(ctx, orgType)
}

func (s *Service) AddOrganization(ctx context.Context, req dto.AddOrganizationRequest) (int64, error) {
	return s.repo.AddOrganization(ctx, req.Name, req.ParentOrganizationID, req.TypeIDs)
}

func (s *Service) EditOrganization(ctx context.Context, id int64, name *string, parentID **int64, typeIDs []int64) error {
	return s.repo.EditOrganization(ctx, id, name, parentID, typeIDs)
}

func (s *Service) DeleteOrganization(ctx context.Context, id int64) error {
	return s.repo.DeleteOrganization(ctx, id)
}

// Organization Types

func (s *Service) AddOrganizationType(ctx context.Context, name string, description *string) (int64, error) {
	return s.repo.AddOrganizationType(ctx, name, description)
}

func (s *Service) GetAllOrganizationTypes(ctx context.Context) ([]organization_type.Model, error) {
	return s.repo.GetAllOrganizationTypes(ctx)
}

func (s *Service) EditOrganizationType(ctx context.Context, id int64, name, description *string) error {
	return s.repo.EditOrganizationType(ctx, id, name, description)
}

func (s *Service) DeleteOrganizationType(ctx context.Context, id string) error {
	return s.repo.DeleteOrganizationType(ctx, id)
}
