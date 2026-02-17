package shutdown

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/shutdown"
	"srmt-admin/internal/lib/service/fileupload"
	"time"
)

type RepoInterface interface {
	AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error)
	EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error
	DeleteShutdown(ctx context.Context, id int64) error
	GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
	GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error)
	MarkShutdownAsViewed(ctx context.Context, id int64) error
	LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error
	UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error
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

func (s *Service) AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error) {
	return s.repo.AddShutdown(ctx, req)
}

func (s *Service) EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error {
	return s.repo.EditShutdown(ctx, id, req)
}

func (s *Service) DeleteShutdown(ctx context.Context, id int64) error {
	return s.repo.DeleteShutdown(ctx, id)
}

func (s *Service) GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error) {
	return s.repo.GetShutdowns(ctx, day)
}

func (s *Service) GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error) {
	return s.repo.GetOrganizationTypesMap(ctx)
}

func (s *Service) MarkShutdownAsViewed(ctx context.Context, id int64) error {
	return s.repo.MarkShutdownAsViewed(ctx, id)
}

func (s *Service) LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error {
	return s.repo.LinkShutdownFiles(ctx, shutdownID, fileIDs)
}

func (s *Service) UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error {
	return s.repo.UnlinkShutdownFiles(ctx, shutdownID)
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
