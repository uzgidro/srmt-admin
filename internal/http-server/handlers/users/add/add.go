package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type newContactRequest struct {
	Name            string     `json:"name" validate:"required"`
	Email           *string    `json:"email,omitempty" validate:"omitempty,email"`
	Phone           *string    `json:"phone,omitempty"`
	IPPhone         *string    `json:"ip_phone,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	ExternalOrgName *string    `json:"external_organization_name,omitempty"`
	OrganizationID  *int64     `json:"organization_id,omitempty"`
	DepartmentID    *int64     `json:"department_id,omitempty"`
	PositionID      *int64     `json:"position_id,omitempty"`
}

// Request - DTO хендлера (гибридный)
type Request struct {
	Login    string  `json:"login" validate:"required"`
	Password string  `json:"password" validate:"required,min=8"`
	Roles    []int64 `json:"role_ids" validate:"required,min=1"`

	// XOR: Либо `contact_id`, либо `contact`
	ContactID *int64             `json:"contact_id,omitempty" validate:"omitempty,gt=0"`
	Contact   *newContactRequest `json:"contact,omitempty" validate:"omitempty"`
}

type Response struct {
	resp.Response
	ID int64 `json:"id"` // ID нового пользователя (из users)
}

type FileUploader interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	DeleteFile(ctx context.Context, objectName string) error
}

type FileMetaSaver interface {
	AddFile(ctx context.Context, fileData file.Model) (int64, error)
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

// UserLinker - интерфейс для репозитория Users
type UserLinker interface {
	AddUser(ctx context.Context, login string, passwordHash []byte, contactID int64) (int64, error)
	IsContactLinked(ctx context.Context, contactID int64) (bool, error)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
	AssignRolesToUser(ctx context.Context, userID int64, roleIDs []int64) error
}

// New - Фабрика хендлера
// (Мы внедряем *два* репозитория/интерфейса)
func New(log *slog.Logger, userRepo UserLinker, uploader FileUploader, fileSaver FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		contentType := r.Header.Get("Content-Type")
		isMultipart := strings.Contains(contentType, "multipart/form-data")

		var req Request
		var iconID *int64

		if isMultipart {
			// Parse multipart form
			const maxUploadSize = 10 * 1024 * 1024 // 10 MB for icon
			r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request or file is too large"))
				return
			}

			// Parse basic fields
			req.Login = r.FormValue("login")
			req.Password = r.FormValue("password")

			// Parse role_ids
			rolesStr := r.FormValue("role_ids")
			if rolesStr != "" {
				rolesStrs := strings.Split(rolesStr, ",")
				for _, roleStr := range rolesStrs {
					roleID, err := strconv.ParseInt(strings.TrimSpace(roleStr), 10, 64)
					if err != nil {
						log.Error("invalid role_id", sl.Err(err), "value", roleStr)
						render.Status(r, http.StatusBadRequest)
						render.JSON(w, r, resp.BadRequest("Invalid role_ids format"))
						return
					}
					req.Roles = append(req.Roles, roleID)
				}
			}

			// Parse contact_id if provided
			if contactIDStr := r.FormValue("contact_id"); contactIDStr != "" {
				contactID, err := strconv.ParseInt(contactIDStr, 10, 64)
				if err != nil {
					log.Error("invalid contact_id", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid contact_id"))
					return
				}
				req.ContactID = &contactID
			}

			// Parse contact object if provided
			if r.FormValue("contact.name") != "" {
				contact := &newContactRequest{
					Name: r.FormValue("contact.name"),
				}
				if email := r.FormValue("contact.email"); email != "" {
					contact.Email = &email
				}
				if phone := r.FormValue("contact.phone"); phone != "" {
					contact.Phone = &phone
				}
				if ipPhone := r.FormValue("contact.ip_phone"); ipPhone != "" {
					contact.IPPhone = &ipPhone
				}
				if dobStr := r.FormValue("contact.dob"); dobStr != "" {
					dob, err := time.Parse(time.DateOnly, dobStr)
					if err != nil {
						log.Error("invalid contact.dob format", sl.Err(err))
						render.Status(r, http.StatusBadRequest)
						render.JSON(w, r, resp.BadRequest("Invalid contact.dob format, use RFC3339"))
						return
					}
					contact.DOB = &dob
				}
				if extOrg := r.FormValue("contact.external_organization_name"); extOrg != "" {
					contact.ExternalOrgName = &extOrg
				}
				if orgIDStr := r.FormValue("contact.organization_id"); orgIDStr != "" {
					orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
					if err != nil {
						log.Error("invalid contact.organization_id", sl.Err(err))
						render.Status(r, http.StatusBadRequest)
						render.JSON(w, r, resp.BadRequest("Invalid contact.organization_id"))
						return
					}
					contact.OrganizationID = &orgID
				}
				if deptIDStr := r.FormValue("contact.department_id"); deptIDStr != "" {
					deptID, err := strconv.ParseInt(deptIDStr, 10, 64)
					if err != nil {
						log.Error("invalid contact.department_id", sl.Err(err))
						render.Status(r, http.StatusBadRequest)
						render.JSON(w, r, resp.BadRequest("Invalid contact.department_id"))
						return
					}
					contact.DepartmentID = &deptID
				}
				if posIDStr := r.FormValue("contact.position_id"); posIDStr != "" {
					posID, err := strconv.ParseInt(posIDStr, 10, 64)
					if err != nil {
						log.Error("invalid contact.position_id", sl.Err(err))
						render.Status(r, http.StatusBadRequest)
						render.JSON(w, r, resp.BadRequest("Invalid contact.position_id"))
						return
					}
					contact.PositionID = &posID
				}
				req.Contact = contact
			}

			// Handle icon file upload if present
			formFile, handler, err := r.FormFile("icon")
			if err == nil {
				defer formFile.Close()

				// Get or create "icon" category
				cat, err := fileSaver.GetCategoryByName(r.Context(), "icon")
				if err != nil {
					log.Error("failed to get icon category", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to process icon"))
					return
				}

				// Generate unique object key
				objectKey := fmt.Sprintf("icon/%s%s",
					uuid.New().String(),
					filepath.Ext(handler.Filename),
				)

				// Upload to storage
				err = uploader.UploadFile(r.Context(), objectKey, formFile, handler.Size, handler.Header.Get("Content-Type"))
				if err != nil {
					log.Error("failed to upload icon to storage", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to upload icon"))
					return
				}

				// Save file metadata
				fileModel := file.Model{
					FileName:   handler.Filename,
					ObjectKey:  objectKey,
					CategoryID: cat.GetID(),
					MimeType:   handler.Header.Get("Content-Type"),
					SizeBytes:  handler.Size,
					CreatedAt:  time.Now(),
				}

				fileID, err := fileSaver.AddFile(r.Context(), fileModel)
				if err != nil {
					log.Error("failed to save icon metadata", sl.Err(err))
					// Compensate: delete uploaded file
					if delErr := uploader.DeleteFile(r.Context(), objectKey); delErr != nil {
						log.Error("compensation failed: could not delete orphaned icon", sl.Err(delErr))
					}
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to save icon"))
					return
				}

				iconID = &fileID
			} else if err != http.ErrMissingFile {
				log.Error("failed to get icon file", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid icon file"))
				return
			}

		} else {
			// Parse JSON
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// --- Логика XOR-валидации ---
		if (req.ContactID == nil && req.Contact == nil) || (req.ContactID != nil && req.Contact != nil) {
			log.Warn("validation failed: must provide either contact_id or contact object")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Must provide either 'contact_id' or 'contact' object, but not both"))
			return
		}

		// --- Хеширование пароля ---
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Error("failed to hash password", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to process request"))
			return
		}

		var newUserID int64

		// --- Сценарий 1: "Привязать" существующий контакт ---
		if req.ContactID != nil {
			contactID := *req.ContactID

			// Проверяем, не привязан ли уже этот контакт
			isLinked, err := userRepo.IsContactLinked(r.Context(), contactID)
			if err != nil {
				log.Error("failed to check contact link", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to check contact"))
				return
			}
			if isLinked {
				log.Warn("contact is already linked to another user", "contact_id", contactID)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.BadRequest("This contact is already linked to a user"))
				return
			}

			// Создаем пользователя
			newUserID, err = userRepo.AddUser(r.Context(), req.Login, passwordHash, contactID)
			if err != nil {
				if errors.Is(err, storage.ErrDuplicate) { // (Unique(login))
					log.Warn("duplicate login", "login", req.Login)
					render.Status(r, http.StatusConflict)
					render.JSON(w, r, resp.BadRequest("Login already exists"))
					return
				}
				if errors.Is(err, storage.ErrForeignKeyViolation) { // (contact_id не найден)
					log.Warn("contact_id not found", "contact_id", contactID)
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Contact not found"))
					return
				}
				log.Error("failed to add user (link)", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to add user"))
				return
			}

			// --- Сценарий 2: "Создать на ходу" ---
			// (Этот сценарий НЕ транзакционный, т.к. `AddContact` и `AddUser` - отдельные вызовы.
			//  Для транзакционности нужен был бы *третий* метод репозитория: `AddUserWithNewContact`)
		} else if req.Contact != nil {
			// 1. Создаем контакт
			storageReq := dto.AddContactRequest{
				Name:            req.Contact.Name,
				Email:           req.Contact.Email,
				Phone:           req.Contact.Phone,
				IPPhone:         req.Contact.IPPhone,
				DOB:             req.Contact.DOB,
				ExternalOrgName: req.Contact.ExternalOrgName,
				IconID:          iconID,
				OrganizationID:  req.Contact.OrganizationID,
				DepartmentID:    req.Contact.DepartmentID,
				PositionID:      req.Contact.PositionID,
			}
			newContactID, err := userRepo.AddContact(r.Context(), storageReq)
			if err != nil {
				// (Обрабатываем ошибки от AddContact)
				if errors.Is(err, storage.ErrDuplicate) {
					log.Warn("duplicate contact data", "name", req.Contact.Name)
					render.Status(r, http.StatusConflict)
					render.JSON(w, r, resp.BadRequest("Contact with this email or phone already exists"))
					return
				}
				if errors.Is(err, storage.ErrForeignKeyViolation) {
					log.Warn("FK violation on contact create", "org_id", req.Contact.OrganizationID)
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid organization, department or position"))
					return
				}
				log.Error("failed to add contact (on-the-fly)", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to create contact part"))
				return
			}

			// 2. Создаем пользователя (привязываем к newContactID)
			newUserID, err = userRepo.AddUser(r.Context(), req.Login, passwordHash, newContactID)
			if err != nil {
				// (Если AddUser падает, `AddContact` НЕ откатывается - это ограничение
				//  данной реализации. Можно добавить "ручной" откат контакта.)
				if errors.Is(err, storage.ErrDuplicate) { // (Unique(login))
					log.Warn("duplicate login", "login", req.Login)
					render.Status(r, http.StatusConflict)
					render.JSON(w, r, resp.BadRequest("Login already exists"))
					return
				}
				log.Error("failed to add user (on-the-fly)", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to add user"))
				return
			}
		}

		if err := userRepo.AssignRolesToUser(r.Context(), newUserID, req.Roles); err != nil {
			log.Info("failed to assign role to user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to assign role to user"))
			return
		}

		log.Info("user added", slog.Int64("id", newUserID))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: newUserID})
	}
}
