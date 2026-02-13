package document

import (
	"context"
	"errors"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/document"
	"srmt-admin/internal/storage"
)

type RepoInterface interface {
	CreateHRDocument(ctx context.Context, req dto.CreateHRDocumentRequest, createdBy int64) (int64, error)
	GetHRDocumentByID(ctx context.Context, id int64) (*document.HRDocument, error)
	GetAllHRDocuments(ctx context.Context, filters dto.HRDocumentFilters) ([]*document.HRDocument, error)
	UpdateHRDocument(ctx context.Context, id int64, req dto.UpdateHRDocumentRequest) error
	DeleteHRDocument(ctx context.Context, id int64) error

	CreateDocumentRequest(ctx context.Context, employeeID int64, req dto.CreateDocumentRequestReq) (int64, error)
	GetAllDocumentRequests(ctx context.Context, employeeID *int64) ([]*document.DocumentRequest, error)
	GetDocumentRequestByID(ctx context.Context, id int64) (*document.DocumentRequest, error)
	UpdateDocumentRequestStatus(ctx context.Context, id int64, status string, rejectionReason *string) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// ==================== Documents ====================

func (s *Service) CreateDocument(ctx context.Context, req dto.CreateHRDocumentRequest, createdBy int64) (int64, error) {
	return s.repo.CreateHRDocument(ctx, req, createdBy)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*document.HRDocument, error) {
	doc, err := s.repo.GetHRDocumentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc.Signatures == nil {
		doc.Signatures = []document.Signature{}
	}
	return doc, nil
}

func (s *Service) GetAll(ctx context.Context, filters dto.HRDocumentFilters) ([]*document.HRDocument, error) {
	docs, err := s.repo.GetAllHRDocuments(ctx, filters)
	if err != nil {
		return nil, err
	}
	if docs == nil {
		docs = []*document.HRDocument{}
	}
	return docs, nil
}

func (s *Service) UpdateDocument(ctx context.Context, id int64, req dto.UpdateHRDocumentRequest) error {
	return s.repo.UpdateHRDocument(ctx, id, req)
}

func (s *Service) DeleteDocument(ctx context.Context, id int64) error {
	doc, err := s.repo.GetHRDocumentByID(ctx, id)
	if err != nil {
		return err
	}
	if doc.Status != "draft" {
		return storage.ErrInvalidStatus
	}
	return s.repo.DeleteHRDocument(ctx, id)
}

func (s *Service) Download(ctx context.Context, id int64) (*document.HRDocument, error) {
	return s.repo.GetHRDocumentByID(ctx, id)
}

// ==================== Document Requests ====================

func (s *Service) CreateRequest(ctx context.Context, employeeID int64, req dto.CreateDocumentRequestReq) (int64, error) {
	return s.repo.CreateDocumentRequest(ctx, employeeID, req)
}

func (s *Service) GetAllRequests(ctx context.Context, employeeID *int64) ([]*document.DocumentRequest, error) {
	requests, err := s.repo.GetAllDocumentRequests(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if requests == nil {
		requests = []*document.DocumentRequest{}
	}
	return requests, nil
}

func (s *Service) ApproveRequest(ctx context.Context, id int64) error {
	req, err := s.repo.GetDocumentRequestByID(ctx, id)
	if err != nil {
		return err
	}
	if req.Status != "pending" {
		return errors.New("request is not in pending status")
	}
	return s.repo.UpdateDocumentRequestStatus(ctx, id, "in_progress", nil)
}

func (s *Service) RejectRequest(ctx context.Context, id int64, reason string) error {
	req, err := s.repo.GetDocumentRequestByID(ctx, id)
	if err != nil {
		return err
	}
	if req.Status != "pending" {
		return errors.New("request is not in pending status")
	}
	return s.repo.UpdateDocumentRequestStatus(ctx, id, "rejected", &reason)
}
