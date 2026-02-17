package signature

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/signature"
)

type RepoInterface interface {
	GetPendingSignatureDocuments(ctx context.Context) ([]signature.PendingDocument, error)
	GetDocumentSignatures(ctx context.Context, docType string, docID int64) ([]signature.Signature, error)
	SignDocument(ctx context.Context, docType string, docID int64, req dto.SignDocumentRequest, userID int64) error
	GetSignedStatusInfo(ctx context.Context) (*dto.StatusInfo, error)
	RejectSignature(ctx context.Context, docType string, docID int64, reason *string, userID int64) error
	GetSignatureRejectedStatusInfo(ctx context.Context) (*dto.StatusInfo, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetPendingSignatureDocuments(ctx context.Context) ([]signature.PendingDocument, error) {
	return s.repo.GetPendingSignatureDocuments(ctx)
}

func (s *Service) GetDocumentSignatures(ctx context.Context, docType string, docID int64) ([]signature.Signature, error) {
	return s.repo.GetDocumentSignatures(ctx, docType, docID)
}

func (s *Service) SignDocument(ctx context.Context, docType string, docID int64, req dto.SignDocumentRequest, userID int64) error {
	return s.repo.SignDocument(ctx, docType, docID, req, userID)
}

func (s *Service) GetSignedStatusInfo(ctx context.Context) (*dto.StatusInfo, error) {
	return s.repo.GetSignedStatusInfo(ctx)
}

func (s *Service) RejectSignature(ctx context.Context, docType string, docID int64, reason *string, userID int64) error {
	return s.repo.RejectSignature(ctx, docType, docID, reason, userID)
}

func (s *Service) GetSignatureRejectedStatusInfo(ctx context.Context) (*dto.StatusInfo, error) {
	return s.repo.GetSignatureRejectedStatusInfo(ctx)
}
