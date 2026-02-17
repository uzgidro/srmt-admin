package legaldocument

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	filemodel "srmt-admin/internal/lib/model/file"
	legal_document "srmt-admin/internal/lib/model/legal-document"
	legal_document_type "srmt-admin/internal/lib/model/legal-document-type"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// Legal document CRUD
	AddLegalDocument(ctx context.Context, req dto.AddLegalDocumentRequest, createdByID int64) (int64, error)
	EditLegalDocument(ctx context.Context, id int64, req dto.EditLegalDocumentRequest, updatedByID int64) error
	DeleteLegalDocument(ctx context.Context, id int64) error
	GetAllLegalDocuments(ctx context.Context, filters dto.GetAllLegalDocumentsFilters) ([]*legal_document.ResponseModel, error)
	GetLegalDocumentByID(ctx context.Context, id int64) (*legal_document.ResponseModel, error)

	// Reference data
	GetAllLegalDocumentTypes(ctx context.Context) ([]legal_document_type.Model, error)

	// File linking
	LinkLegalDocumentFiles(ctx context.Context, documentID int64, fileIDs []int64) error
	UnlinkLegalDocumentFiles(ctx context.Context, documentID int64) error

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

func (s *Service) AddLegalDocument(ctx context.Context, req dto.AddLegalDocumentRequest, createdByID int64) (int64, error) {
	return s.repo.AddLegalDocument(ctx, req, createdByID)
}

func (s *Service) EditLegalDocument(ctx context.Context, id int64, req dto.EditLegalDocumentRequest, updatedByID int64) error {
	return s.repo.EditLegalDocument(ctx, id, req, updatedByID)
}

func (s *Service) DeleteLegalDocument(ctx context.Context, id int64) error {
	return s.repo.DeleteLegalDocument(ctx, id)
}

func (s *Service) GetAllLegalDocuments(ctx context.Context, filters dto.GetAllLegalDocumentsFilters) ([]*legal_document.ResponseModel, error) {
	return s.repo.GetAllLegalDocuments(ctx, filters)
}

func (s *Service) GetLegalDocumentByID(ctx context.Context, id int64) (*legal_document.ResponseModel, error) {
	return s.repo.GetLegalDocumentByID(ctx, id)
}

func (s *Service) GetAllLegalDocumentTypes(ctx context.Context) ([]legal_document_type.Model, error) {
	return s.repo.GetAllLegalDocumentTypes(ctx)
}

func (s *Service) LinkLegalDocumentFiles(ctx context.Context, documentID int64, fileIDs []int64) error {
	return s.repo.LinkLegalDocumentFiles(ctx, documentID, fileIDs)
}

func (s *Service) UnlinkLegalDocumentFiles(ctx context.Context, documentID int64) error {
	return s.repo.UnlinkLegalDocumentFiles(ctx, documentID)
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
