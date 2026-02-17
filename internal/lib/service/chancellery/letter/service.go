package letter

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/letter"
	letter_type "srmt-admin/internal/lib/model/letter-type"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// Letter CRUD
	AddLetter(ctx context.Context, req dto.AddLetterRequest, createdByID int64) (int64, error)
	EditLetter(ctx context.Context, id int64, req dto.EditLetterRequest, updatedByID int64) error
	DeleteLetter(ctx context.Context, id int64) error
	GetAllLetters(ctx context.Context, filters dto.GetAllLettersFilters) ([]*letter.ResponseModel, error)
	GetLetterByID(ctx context.Context, id int64) (*letter.ResponseModel, error)

	// Status workflow
	GetLetterStatusHistory(ctx context.Context, letterID int64) ([]letter.StatusHistory, error)
	AddLetterStatusHistoryComment(ctx context.Context, letterID int64, comment string) error

	// Reference data
	GetAllLetterTypes(ctx context.Context) ([]letter_type.Model, error)

	// File linking
	LinkLetterFiles(ctx context.Context, letterID int64, fileIDs []int64) error
	UnlinkLetterFiles(ctx context.Context, letterID int64) error

	// Document linking
	LinkLetterDocuments(ctx context.Context, letterID int64, links []dto.LinkedDocumentRequest, userID int64) error
	UnlinkLetterDocuments(ctx context.Context, letterID int64) error

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

func (s *Service) AddLetter(ctx context.Context, req dto.AddLetterRequest, createdByID int64) (int64, error) {
	return s.repo.AddLetter(ctx, req, createdByID)
}

func (s *Service) EditLetter(ctx context.Context, id int64, req dto.EditLetterRequest, updatedByID int64) error {
	return s.repo.EditLetter(ctx, id, req, updatedByID)
}

func (s *Service) DeleteLetter(ctx context.Context, id int64) error {
	return s.repo.DeleteLetter(ctx, id)
}

func (s *Service) GetAllLetters(ctx context.Context, filters dto.GetAllLettersFilters) ([]*letter.ResponseModel, error) {
	return s.repo.GetAllLetters(ctx, filters)
}

func (s *Service) GetLetterByID(ctx context.Context, id int64) (*letter.ResponseModel, error) {
	return s.repo.GetLetterByID(ctx, id)
}

func (s *Service) GetLetterStatusHistory(ctx context.Context, letterID int64) ([]letter.StatusHistory, error) {
	return s.repo.GetLetterStatusHistory(ctx, letterID)
}

func (s *Service) AddLetterStatusHistoryComment(ctx context.Context, letterID int64, comment string) error {
	return s.repo.AddLetterStatusHistoryComment(ctx, letterID, comment)
}

func (s *Service) GetAllLetterTypes(ctx context.Context) ([]letter_type.Model, error) {
	return s.repo.GetAllLetterTypes(ctx)
}

func (s *Service) LinkLetterFiles(ctx context.Context, letterID int64, fileIDs []int64) error {
	return s.repo.LinkLetterFiles(ctx, letterID, fileIDs)
}

func (s *Service) UnlinkLetterFiles(ctx context.Context, letterID int64) error {
	return s.repo.UnlinkLetterFiles(ctx, letterID)
}

func (s *Service) LinkLetterDocuments(ctx context.Context, letterID int64, links []dto.LinkedDocumentRequest, userID int64) error {
	return s.repo.LinkLetterDocuments(ctx, letterID, links, userID)
}

func (s *Service) UnlinkLetterDocuments(ctx context.Context, letterID int64) error {
	return s.repo.UnlinkLetterDocuments(ctx, letterID)
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
