package file

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/lib/model/file"
)

type RepoInterface interface {
	AddFile(ctx context.Context, fileData file.Model) (int64, error)
	GetFileByID(ctx context.Context, id int64) (file.Model, error)
	DeleteFile(ctx context.Context, id int64) error
	GetCategoryByID(ctx context.Context, id int64) (category.Model, error)
	AddCategory(ctx context.Context, cat category.Model) (int64, error)
	GetAllCategories(ctx context.Context) ([]category.Model, error)
	GetLatestFiles(ctx context.Context) ([]file.LatestFile, error)
	GetLatestFileByCategoryAndDate(ctx context.Context, categoryName string, targetDate string) (file.Model, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddFile(ctx context.Context, fileData file.Model) (int64, error) {
	return s.repo.AddFile(ctx, fileData)
}

func (s *Service) GetFileByID(ctx context.Context, id int64) (file.Model, error) {
	return s.repo.GetFileByID(ctx, id)
}

func (s *Service) DeleteFile(ctx context.Context, id int64) error {
	return s.repo.DeleteFile(ctx, id)
}

func (s *Service) GetCategoryByID(ctx context.Context, id int64) (category.Model, error) {
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *Service) AddCategory(ctx context.Context, cat category.Model) (int64, error) {
	return s.repo.AddCategory(ctx, cat)
}

func (s *Service) GetAllCategories(ctx context.Context) ([]category.Model, error) {
	return s.repo.GetAllCategories(ctx)
}

func (s *Service) GetLatestFiles(ctx context.Context) ([]file.LatestFile, error) {
	return s.repo.GetLatestFiles(ctx)
}

func (s *Service) GetLatestFileByCategoryAndDate(ctx context.Context, categoryName string, targetDate string) (file.Model, error) {
	return s.repo.GetLatestFileByCategoryAndDate(ctx, categoryName, targetDate)
}
