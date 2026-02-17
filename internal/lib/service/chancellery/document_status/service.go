package document_status

import (
	"context"
	"log/slog"
	document_status "srmt-admin/internal/lib/model/document-status"
)

type RepoInterface interface {
	GetAllDocumentStatuses(ctx context.Context) ([]document_status.Model, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetAllDocumentStatuses(ctx context.Context) ([]document_status.Model, error) {
	return s.repo.GetAllDocumentStatuses(ctx)
}
