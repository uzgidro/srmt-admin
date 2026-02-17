package discharge

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/model/discharge"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error)
	EditDischarge(ctx context.Context, id, approvedByID int64, startTime, endTime *time.Time, flowRate *float64, reason *string, approved *bool, organizationID *int64) error
	DeleteDischarge(ctx context.Context, id int64) error
	GetCurrentDischarges(ctx context.Context) ([]discharge.Model, error)
	GetAllDischarges(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Model, error)
	GetDischargesByCascades(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Cascade, error)
	LinkDischargeFiles(ctx context.Context, dischargeID int64, fileIDs []int64) error
	UnlinkDischargeFiles(ctx context.Context, dischargeID int64) error
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

func (s *Service) AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error) {
	return s.repo.AddDischarge(ctx, orgID, createdByID, startTime, endTime, flowRate, reason)
}

func (s *Service) EditDischarge(ctx context.Context, id, approvedByID int64, startTime, endTime *time.Time, flowRate *float64, reason *string, approved *bool, organizationID *int64) error {
	return s.repo.EditDischarge(ctx, id, approvedByID, startTime, endTime, flowRate, reason, approved, organizationID)
}

func (s *Service) DeleteDischarge(ctx context.Context, id int64) error {
	return s.repo.DeleteDischarge(ctx, id)
}

func (s *Service) GetCurrentDischarges(ctx context.Context) ([]discharge.Model, error) {
	return s.repo.GetCurrentDischarges(ctx)
}

func (s *Service) GetAllDischarges(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Model, error) {
	return s.repo.GetAllDischarges(ctx, isOngoing, startDate, endDate)
}

func (s *Service) GetDischargesByCascades(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Cascade, error) {
	return s.repo.GetDischargesByCascades(ctx, isOngoing, startDate, endDate)
}

func (s *Service) LinkDischargeFiles(ctx context.Context, dischargeID int64, fileIDs []int64) error {
	return s.repo.LinkDischargeFiles(ctx, dischargeID, fileIDs)
}

func (s *Service) UnlinkDischargeFiles(ctx context.Context, dischargeID int64) error {
	return s.repo.UnlinkDischargeFiles(ctx, dischargeID)
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
