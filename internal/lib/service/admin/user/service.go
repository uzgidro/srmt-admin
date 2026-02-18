package user

import (
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/contact"
	filemodel "srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/lib/service/fileupload"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	DeleteFile(ctx context.Context, id int64) error // Needed for compensation
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

type Service struct {
	repo     RepoInterface
	uploader fileupload.FileUploader
	log      *slog.Logger
}

func NewService(repo RepoInterface, uploader fileupload.FileUploader, log *slog.Logger) *Service {
	return &Service{repo: repo, uploader: uploader, log: log}
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

// CreateUser handles user creation logic including validation, hashing, file upload, and contact linking
func (s *Service) CreateUser(ctx context.Context, req dto.CreateUserRequest, iconFile *multipart.FileHeader) (int64, error) {
	const op = "service.user.CreateUser"
	log := s.log.With(slog.String("op", op), slog.String("login", req.Login))

	// 1. Validation (XOR check)
	if (req.ContactID == nil && req.Contact == nil) || (req.ContactID != nil && req.Contact != nil) {
		return 0, fmt.Errorf("validation failed: must provide either contact_id or contact object")
	}

	// 2. Hash Password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to hash password: %w", op, err)
	}

	// 3. Handle Icon Upload (if present)
	var iconID *int64
	var uploadedFiles []fileupload.UploadedFileInfo

	if iconFile != nil && req.Contact != nil { // Icon is only relevant if creating a NEW contact
		// Get or create "icon" category
		cat, err := s.repo.GetCategoryByName(ctx, "icon")
		if err != nil {
			return 0, fmt.Errorf("%s: failed to get icon category: %w", op, err)
		}

		// Upload file
		fileInfo, err := fileupload.UploadFileHeader(
			ctx,
			log,
			s.uploader,
			s.repo, // Repo satisfies FileMetaSaver (AddFile, DeleteFile)
			iconFile,
			"icon",
			cat.GetID(),
			time.Now(),
		)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to upload icon: %w", op, err)
		}

		iconID = &fileInfo.ID
		uploadedFiles = append(uploadedFiles, *fileInfo)
	}

	// Setup compensation for file upload (Validation/DB errors after upload)
	defer func() {
		if err != nil {
			// If errors occurred, rollback uploads
			fileupload.CompensateEntityUpload(
				ctx,
				log,
				s.uploader,
				s.repo,
				&fileupload.UploadResult{UploadedFiles: uploadedFiles}, // Helper struct for compensation
			)
		}
	}()

	var newUserID int64

	// 4. Handle Contact (Link or Create)
	if req.ContactID != nil {
		contactID := *req.ContactID
		// Check linkage
		isLinked, err := s.repo.IsContactLinked(ctx, contactID)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to check contact link: %w", op, err)
		}
		if isLinked {
			return 0, fmt.Errorf("contact is already linked to another user")
		}

		// Create User linked to existing contact
		newUserID, err = s.repo.AddUser(ctx, req.Login, passwordHash, contactID)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to add user (link): %w", op, err)
		}

	} else if req.Contact != nil {
		// Create new contact
		contactReq := *req.Contact
		contactReq.IconID = iconID // Set the uploaded icon ID

		contactID, err := s.repo.AddContact(ctx, contactReq)
		if err != nil {
			return 0, fmt.Errorf("%s: failed to create contact: %w", op, err)
		}

		// Create User linked to new contact
		newUserID, err = s.repo.AddUser(ctx, req.Login, passwordHash, contactID)
		if err != nil {
			// Note: If AddUser fails, the created Contact remains (orphan).
			// Ideally, this should be in a transaction. For now, we accept this limitation or add manual rollback of contact.
			// (Adding simple rollback for contact here would be checking repo for DeleteContact)
			// For this refactoring, we stick to current behavior but encapsulated.
			return 0, fmt.Errorf("%s: failed to add user (new contact): %w", op, err)
		}
	}

	// 5. Assign Roles
	if len(req.RoleIDs) > 0 {
		if err := s.repo.AssignRolesToUser(ctx, newUserID, req.RoleIDs); err != nil {
			// If role assignment fails, we should probably rollback user creation?
			// The original handler didn't rollback. Keeping as is for now, but logging error.
			// Ideally: Transaction.
			log.Error("failed to assign roles", sl.Err(err))
			// We return error so the compensation triggers for files?
			// But User is already created. This is a partial failure.
			return 0, fmt.Errorf("%s: failed to assign roles: %w", op, err)
		}
	}

	return newUserID, nil
}

// UpdateUser handles user updates including password, roles, and icon
func (s *Service) UpdateUser(ctx context.Context, userID int64, req dto.UpdateUserRequest, iconFile *multipart.FileHeader) error {
	const op = "service.user.UpdateUser"
	log := s.log.With(slog.String("op", op), slog.Int64("user_id", userID))

	// 1. Hash Password if provided
	var passwordHash []byte
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("%s: failed to hash password: %w", op, err)
		}
		passwordHash = hash
	}

	// 2. Update User (Login, Password, IsActive)
	// Map to Repo DTO
	// Note: Repo expects EditUserRequest which doesn't have Password (passed as arg)
	repoReq := dto.EditUserRequest{
		Login:    req.Login,
		IsActive: req.IsActive,
	}

	if err := s.repo.EditUser(ctx, userID, passwordHash, repoReq); err != nil {
		return fmt.Errorf("%s: failed to edit user: %w", op, err)
	}

	// 3. Update Roles if provided
	if req.RoleIDs != nil {
		if err := s.repo.ReplaceUserRoles(ctx, userID, *req.RoleIDs); err != nil {
			return fmt.Errorf("%s: failed to replace user roles: %w", op, err)
		}
	}

	// 4. Update Icon if provided
	if iconFile != nil {
		// Get or create "icon" category
		cat, err := s.repo.GetCategoryByName(ctx, "icon")
		if err != nil {
			return fmt.Errorf("%s: failed to get icon category: %w", op, err)
		}

		// Upload file
		fileInfo, err := fileupload.UploadFileHeader(
			ctx,
			log,
			s.uploader,
			s.repo,
			iconFile,
			"icon",
			cat.GetID(),
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("%s: failed to upload icon: %w", op, err)
		}

		// Get user to find contact_id
		userModel, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			// Compensation: delete uploaded file
			// We can use same compensation logic as CreateUser or just direct delete
			_ = s.uploader.DeleteFile(ctx, fileInfo.ObjectKey)
			_ = s.repo.DeleteFile(ctx, fileInfo.ID)
			return fmt.Errorf("%s: failed to get user for icon update: %w", op, err)
		}

		// Update contact with new icon
		contactReq := dto.EditContactRequest{
			IconID: &fileInfo.ID,
		}
		if err := s.repo.EditContact(ctx, userModel.ContactID, contactReq); err != nil {
			// Compensation
			_ = s.uploader.DeleteFile(ctx, fileInfo.ObjectKey)
			_ = s.repo.DeleteFile(ctx, fileInfo.ID)
			return fmt.Errorf("%s: failed to update contact icon: %w", op, err)
		}
	}

	return nil
}
