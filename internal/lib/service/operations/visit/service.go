package visit

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/visit"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	AddVisit(ctx context.Context, req dto.AddVisitRequest) (int64, error)
	EditVisit(ctx context.Context, id int64, req dto.EditVisitRequest) error
	DeleteVisit(ctx context.Context, id int64) error
	GetVisits(ctx context.Context, day time.Time) ([]*visit.ResponseModel, error)
	LinkVisitFiles(ctx context.Context, visitID int64, fileIDs []int64) error
	UnlinkVisitFiles(ctx context.Context, visitID int64) error
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

func (s *Service) AddVisit(ctx context.Context, req dto.AddVisitRequest) (int64, error) {
	return s.repo.AddVisit(ctx, req)
}

func (s *Service) EditVisit(ctx context.Context, id int64, req dto.EditVisitRequest) error {
	return s.repo.EditVisit(ctx, id, req)
}

func (s *Service) DeleteVisit(ctx context.Context, id int64) error {
	return s.repo.DeleteVisit(ctx, id)
}

func (s *Service) GetVisits(ctx context.Context, day time.Time) ([]*visit.ResponseModel, error) {
	return s.repo.GetVisits(ctx, day)
}

func (s *Service) LinkVisitFiles(ctx context.Context, visitID int64, fileIDs []int64) error {
	return s.repo.LinkVisitFiles(ctx, visitID, fileIDs)
}

func (s *Service) UnlinkVisitFiles(ctx context.Context, visitID int64) error {
	return s.repo.UnlinkVisitFiles(ctx, visitID)
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
