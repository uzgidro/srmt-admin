package decree

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/decree"
	decree_type "srmt-admin/internal/lib/model/decree-type"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// Decree CRUD
	AddDecree(ctx context.Context, req dto.AddDecreeRequest, createdByID int64) (int64, error)
	EditDecree(ctx context.Context, id int64, req dto.EditDecreeRequest, updatedByID int64) error
	DeleteDecree(ctx context.Context, id int64) error
	GetAllDecrees(ctx context.Context, filters dto.GetAllDecreesFilters) ([]*decree.ResponseModel, error)
	GetDecreeByID(ctx context.Context, id int64) (*decree.ResponseModel, error)

	// Status workflow
	GetDecreeStatusHistory(ctx context.Context, decreeID int64) ([]decree.StatusHistory, error)
	AddDecreeStatusHistoryComment(ctx context.Context, decreeID int64, comment string) error

	// Reference data
	GetAllDecreeTypes(ctx context.Context) ([]decree_type.Model, error)

	// File linking
	LinkDecreeFiles(ctx context.Context, decreeID int64, fileIDs []int64) error
	UnlinkDecreeFiles(ctx context.Context, decreeID int64) error

	// Document linking
	LinkDecreeDocuments(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error
	UnlinkDecreeDocuments(ctx context.Context, decreeID int64) error

	// File metadata
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

func (s *Service) AddDecree(ctx context.Context, req dto.AddDecreeRequest, createdByID int64) (int64, error) {
	return s.repo.AddDecree(ctx, req, createdByID)
}

func (s *Service) EditDecree(ctx context.Context, id int64, req dto.EditDecreeRequest, updatedByID int64) error {
	return s.repo.EditDecree(ctx, id, req, updatedByID)
}

func (s *Service) DeleteDecree(ctx context.Context, id int64) error {
	return s.repo.DeleteDecree(ctx, id)
}

func (s *Service) GetAllDecrees(ctx context.Context, filters dto.GetAllDecreesFilters) ([]*decree.ResponseModel, error) {
	return s.repo.GetAllDecrees(ctx, filters)
}

func (s *Service) GetDecreeByID(ctx context.Context, id int64) (*decree.ResponseModel, error) {
	return s.repo.GetDecreeByID(ctx, id)
}

func (s *Service) GetDecreeStatusHistory(ctx context.Context, decreeID int64) ([]decree.StatusHistory, error) {
	return s.repo.GetDecreeStatusHistory(ctx, decreeID)
}

func (s *Service) AddDecreeStatusHistoryComment(ctx context.Context, decreeID int64, comment string) error {
	return s.repo.AddDecreeStatusHistoryComment(ctx, decreeID, comment)
}

func (s *Service) GetAllDecreeTypes(ctx context.Context) ([]decree_type.Model, error) {
	return s.repo.GetAllDecreeTypes(ctx)
}

func (s *Service) LinkDecreeFiles(ctx context.Context, decreeID int64, fileIDs []int64) error {
	return s.repo.LinkDecreeFiles(ctx, decreeID, fileIDs)
}

func (s *Service) UnlinkDecreeFiles(ctx context.Context, decreeID int64) error {
	return s.repo.UnlinkDecreeFiles(ctx, decreeID)
}

func (s *Service) LinkDecreeDocuments(ctx context.Context, decreeID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	return s.repo.LinkDecreeDocuments(ctx, decreeID, links, userID)
}

func (s *Service) UnlinkDecreeDocuments(ctx context.Context, decreeID int64) error {
	return s.repo.UnlinkDecreeDocuments(ctx, decreeID)
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
