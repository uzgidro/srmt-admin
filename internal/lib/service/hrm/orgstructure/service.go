package orgstructure

import (
	"context"
	"errors"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/orgstructure"
)

type RepoInterface interface {
	CreateOrgUnit(ctx context.Context, req dto.CreateOrgUnitRequest) (int64, error)
	GetOrgUnitByID(ctx context.Context, id int64) (*orgstructure.OrgUnit, error)
	GetAllOrgUnits(ctx context.Context) ([]*orgstructure.OrgUnit, error)
	UpdateOrgUnit(ctx context.Context, id int64, req dto.UpdateOrgUnitRequest) error
	DeleteOrgUnit(ctx context.Context, id int64) error
	HasChildOrgUnits(ctx context.Context, id int64) (bool, error)
	GetUnitEmployees(ctx context.Context, unitID int64) ([]*orgstructure.OrgEmployee, error)
	GetAllOrgEmployees(ctx context.Context) ([]*orgstructure.OrgEmployee, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ==================== Org Units ====================

func (s *Service) GetTree(ctx context.Context) ([]orgstructure.OrgUnit, error) {
	units, err := s.repo.GetAllOrgUnits(ctx)
	if err != nil {
		return nil, err
	}
	if units == nil {
		return []orgstructure.OrgUnit{}, nil
	}

	// Build tree in memory
	unitMap := make(map[int64]*orgstructure.OrgUnit)
	for _, u := range units {
		unitMap[u.ID] = u
	}

	var roots []orgstructure.OrgUnit
	for _, u := range units {
		if u.ParentID == nil {
			roots = append(roots, *u)
		} else {
			if parent, ok := unitMap[*u.ParentID]; ok {
				parent.Children = append(parent.Children, *u)
			} else {
				roots = append(roots, *u)
			}
		}
	}

	if roots == nil {
		roots = []orgstructure.OrgUnit{}
	}

	return roots, nil
}

func (s *Service) Create(ctx context.Context, req dto.CreateOrgUnitRequest) (int64, error) {
	return s.repo.CreateOrgUnit(ctx, req)
}

func (s *Service) Update(ctx context.Context, id int64, req dto.UpdateOrgUnitRequest) error {
	return s.repo.UpdateOrgUnit(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	hasChildren, err := s.repo.HasChildOrgUnits(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return errors.New("cannot delete unit with children")
	}
	return s.repo.DeleteOrgUnit(ctx, id)
}

// ==================== Employees ====================

func (s *Service) GetUnitEmployees(ctx context.Context, unitID int64) ([]*orgstructure.OrgEmployee, error) {
	employees, err := s.repo.GetUnitEmployees(ctx, unitID)
	if err != nil {
		return nil, err
	}
	if employees == nil {
		employees = []*orgstructure.OrgEmployee{}
	}
	return employees, nil
}

func (s *Service) GetAllEmployees(ctx context.Context) ([]*orgstructure.OrgEmployee, error) {
	employees, err := s.repo.GetAllOrgEmployees(ctx)
	if err != nil {
		return nil, err
	}
	if employees == nil {
		employees = []*orgstructure.OrgEmployee{}
	}
	return employees, nil
}
