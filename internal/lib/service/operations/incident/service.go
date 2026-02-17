package incident

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	AddIncident(ctx context.Context, orgID *int64, incidentTime time.Time, description string, createdByID int64) (int64, error)
	EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest) error
	DeleteIncident(ctx context.Context, id int64) error
	GetIncidents(ctx context.Context, day time.Time) ([]*incident.ResponseModel, error)
	LinkIncidentFiles(ctx context.Context, incidentID int64, fileIDs []int64) error
	UnlinkIncidentFiles(ctx context.Context, incidentID int64) error
	AddFile(ctx context.Context, fileData filemodel.Model) (int64, error)
	DeleteFile(ctx context.Context, id int64) error
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddIncident(ctx context.Context, orgID *int64, incidentTime time.Time, description string, createdByID int64) (int64, error) {
	return s.repo.AddIncident(ctx, orgID, incidentTime, description, createdByID)
}

func (s *Service) EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest) error {
	return s.repo.EditIncident(ctx, id, req)
}

func (s *Service) DeleteIncident(ctx context.Context, id int64) error {
	return s.repo.DeleteIncident(ctx, id)
}

func (s *Service) GetIncidents(ctx context.Context, day time.Time) ([]*incident.ResponseModel, error) {
	return s.repo.GetIncidents(ctx, day)
}

func (s *Service) LinkIncidentFiles(ctx context.Context, incidentID int64, fileIDs []int64) error {
	return s.repo.LinkIncidentFiles(ctx, incidentID, fileIDs)
}

func (s *Service) UnlinkIncidentFiles(ctx context.Context, incidentID int64) error {
	return s.repo.UnlinkIncidentFiles(ctx, incidentID)
}

func (s *Service) AddFile(ctx context.Context, fileData filemodel.Model) (int64, error) {
	return s.repo.AddFile(ctx, fileData)
}

func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	return s.repo.DeleteFile(ctx, id)
}

func (s *Service) GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error) {
	return s.repo.GetCategoryByName(ctx, categoryName)
}
