package contact

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest) error
	DeleteContact(ctx context.Context, id int64) error
	GetAllContacts(ctx context.Context, filters dto.GetAllContactsFilters) ([]*contact.Model, error)
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)
	AddFile(ctx context.Context, fileData filemodel.Model) (int64, error)
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error) {
	return s.repo.AddContact(ctx, req)
}

func (s *Service) EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest) error {
	return s.repo.EditContact(ctx, contactID, req)
}

func (s *Service) DeleteContact(ctx context.Context, id int64) error {
	return s.repo.DeleteContact(ctx, id)
}

func (s *Service) GetAllContacts(ctx context.Context, filters dto.GetAllContactsFilters) ([]*contact.Model, error) {
	return s.repo.GetAllContacts(ctx, filters)
}

func (s *Service) GetContactByID(ctx context.Context, id int64) (*contact.Model, error) {
	return s.repo.GetContactByID(ctx, id)
}

func (s *Service) AddFile(ctx context.Context, fileData filemodel.Model) (int64, error) {
	return s.repo.AddFile(ctx, fileData)
}

func (s *Service) GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error) {
	return s.repo.GetCategoryByName(ctx, categoryName)
}
