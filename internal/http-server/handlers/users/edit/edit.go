package edit

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
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	Login    *string `json:"login,omitempty" validate:"omitempty,min=1"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=8"`
	IsActive *bool   `json:"is_active,omitempty"`
	RoleIDs  []int64 `json:"role_ids,omitempty"`
}

type FileUploader interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	DeleteFile(ctx context.Context, objectName string) error
}

type FileMetaSaver interface {
	AddFile(ctx context.Context, fileData file.Model) (int64, error)
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

// UserUpdater - интерфейс репозитория (использует DTO из storage)
type UserUpdater interface {
	EditUser(ctx context.Context, userID int64, passwordHash []byte, req dto.EditUserRequest) error
	ReplaceUserRoles(ctx context.Context, userID int64, roleIDs []int64) error
	GetUserByID(ctx context.Context, id int64) (*user.Model, error)
	EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest) error
}

func New(log *slog.Logger, updater UserUpdater, uploader FileUploader, fileSaver FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.update.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Получаем ID из URL
		idStr := chi.URLParam(r, "userID")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'userID' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'userID' parameter"))
			return
		}

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

			// Parse form fields (all optional for PATCH)
			if login := r.FormValue("login"); login != "" {
				req.Login = &login
			}
			if password := r.FormValue("password"); password != "" {
				req.Password = &password
			}
			if isActiveStr := r.FormValue("is_active"); isActiveStr != "" {
				isActive, err := strconv.ParseBool(isActiveStr)
				if err != nil {
					log.Error("invalid is_active", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid is_active value"))
					return
				}
				req.IsActive = &isActive
			}

			// Parse role_ids
			if rolesStr := r.FormValue("role_ids"); rolesStr != "" {
				rolesStrs := strings.Split(rolesStr, ",")
				for _, roleStr := range rolesStrs {
					roleID, err := strconv.ParseInt(strings.TrimSpace(roleStr), 10, 64)
					if err != nil {
						log.Error("invalid role_id", sl.Err(err), "value", roleStr)
						render.Status(r, http.StatusBadRequest)
						render.JSON(w, r, resp.BadRequest("Invalid role_ids format"))
						return
					}
					req.RoleIDs = append(req.RoleIDs, roleID)
				}
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
			// 2. Декодируем JSON
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}
		}

		// 3. Валидация DTO
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// 4. (Опциональное) Хеширование пароля
		var passwordHash []byte
		if req.Password != nil {
			hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if err != nil {
				log.Error("failed to hash password", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to process request"))
				return
			}
			passwordHash = hash
		}

		// 5. Маппинг в DTO хранилища
		storageReq := dto.EditUserRequest{
			Login:    req.Login,
			IsActive: req.IsActive,
		}

		// 6. Вызываем репозиторий
		err = updater.EditUser(r.Context(), id, passwordHash, storageReq)
		if err != nil {
			// 7. Обработка ошибок
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("user not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("User not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("duplicate login on update")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.BadRequest("Login already exists"))
				return
			}
			log.Error("failed to update user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update user"))
			return
		}

		// 7. Replace user roles if role_ids is provided
		if req.RoleIDs != nil {
			err = updater.ReplaceUserRoles(r.Context(), id, req.RoleIDs)
			if err != nil {
				if errors.Is(err, storage.ErrForeignKeyViolation) {
					log.Warn("invalid role_id in list", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("One or more role IDs are invalid"))
					return
				}
				log.Error("failed to replace user roles", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to update user roles"))
				return
			}
			log.Info("user roles replaced", slog.Int64("user_id", id), slog.Int("role_count", len(req.RoleIDs)))
		}

		// 8. Update contact icon if uploaded
		if iconID != nil {
			// Get user to find contact_id
			userModel, err := updater.GetUserByID(r.Context(), id)
			if err != nil {
				log.Error("failed to get user for contact update", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to update icon"))
				return
			}

			// Update contact with new icon
			contactReq := dto.EditContactRequest{
				IconID: iconID,
			}
			err = updater.EditContact(r.Context(), userModel.ContactID, contactReq)
			if err != nil {
				log.Error("failed to update contact icon", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to update icon"))
				return
			}
			log.Info("contact icon updated", slog.Int64("contact_id", userModel.ContactID))
		}

		log.Info("user updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
