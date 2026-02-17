package user

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/contact"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/lib/service/fileupload"
)

type RepoInterface interface {
	// User CRUD
	AddUser(ctx context.Context, login string, passwordHash []byte, contactID int64) (int64, error)
	EditUser(ctx context.Context, userID int64, passwordHash []byte, req dto.EditUserRequest) error
	DeleteUser(ctx context.Context, id int64) error
	GetAllUsers(ctx context.Context, filters dto.GetAllUsersFilters) ([]*user.Model, error)
	GetUserByID(ctx context.Context, id int64) (*user.Model, error)
	IsContactLinked(ctx context.Context, contactID int64) (bool, error)

	// Contact operations (for inline contact creation on user add, icon update on user edit)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)
	EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest) error

	// Role operations
	AssignRolesToUser(ctx context.Context, userID int64, roleIDs []int64) error
	RevokeRole(ctx context.Context, userID, roleID int64) error
	ReplaceUserRoles(ctx context.Context, userID int64, roleIDs []int64) error

	// File metadata (for icon upload)
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

func (s *Service) AddUser(ctx context.Context, login string, passwordHash []byte, contactID int64) (int64, error) {
	return s.repo.AddUser(ctx, login, passwordHash, contactID)
}

func (s *Service) EditUser(ctx context.Context, userID int64, passwordHash []byte, req dto.EditUserRequest) error {
	return s.repo.EditUser(ctx, userID, passwordHash, req)
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *Service) GetAllUsers(ctx context.Context, filters dto.GetAllUsersFilters) ([]*user.Model, error) {
	return s.repo.GetAllUsers(ctx, filters)
}

func (s *Service) GetUserByID(ctx context.Context, id int64) (*user.Model, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) IsContactLinked(ctx context.Context, contactID int64) (bool, error) {
	return s.repo.IsContactLinked(ctx, contactID)
}

func (s *Service) AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error) {
	return s.repo.AddContact(ctx, req)
}

func (s *Service) GetContactByID(ctx context.Context, id int64) (*contact.Model, error) {
	return s.repo.GetContactByID(ctx, id)
}

func (s *Service) EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest) error {
	return s.repo.EditContact(ctx, contactID, req)
}

func (s *Service) AssignRolesToUser(ctx context.Context, userID int64, roleIDs []int64) error {
	return s.repo.AssignRolesToUser(ctx, userID, roleIDs)
}

func (s *Service) RevokeRole(ctx context.Context, userID, roleID int64) error {
	return s.repo.RevokeRole(ctx, userID, roleID)
}

func (s *Service) ReplaceUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	return s.repo.ReplaceUserRoles(ctx, userID, roleIDs)
}

func (s *Service) AddFile(ctx context.Context, fileData filemodel.Model) (int64, error) {
	return s.repo.AddFile(ctx, fileData)
}

func (s *Service) GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error) {
	return s.repo.GetCategoryByName(ctx, categoryName)
}
