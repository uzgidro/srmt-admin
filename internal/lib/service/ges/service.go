package ges

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/lib/model/visit"
	"time"
)

type RepoInterface interface {
	GetOrganizationByID(ctx context.Context, id int64) (*organization.Model, error)
	GetDepartmentsByOrgID(ctx context.Context, orgID int64) ([]*department.Model, error)
	GetContactsByOrgID(ctx context.Context, orgID int64) ([]*contact.Model, error)
	GetShutdownsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error)
	GetDischargesByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]discharge.Model, error)
	GetIncidentsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*incident.ResponseModel, error)
	GetVisitsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*visit.ResponseModel, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetOrganizationByID(ctx context.Context, id int64) (*organization.Model, error) {
	return s.repo.GetOrganizationByID(ctx, id)
}

func (s *Service) GetDepartmentsByOrgID(ctx context.Context, orgID int64) ([]*department.Model, error) {
	return s.repo.GetDepartmentsByOrgID(ctx, orgID)
}

func (s *Service) GetContactsByOrgID(ctx context.Context, orgID int64) ([]*contact.Model, error) {
	return s.repo.GetContactsByOrgID(ctx, orgID)
}

func (s *Service) GetShutdownsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error) {
	return s.repo.GetShutdownsByOrgID(ctx, orgID, startDate, endDate)
}

func (s *Service) GetDischargesByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]discharge.Model, error) {
	return s.repo.GetDischargesByOrgID(ctx, orgID, startDate, endDate)
}

func (s *Service) GetIncidentsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*incident.ResponseModel, error) {
	return s.repo.GetIncidentsByOrgID(ctx, orgID, startDate, endDate)
}

func (s *Service) GetVisitsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*visit.ResponseModel, error) {
	return s.repo.GetVisitsByOrgID(ctx, orgID, startDate, endDate)
}
