package edit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	Login    *string `json:"login,omitempty" validate:"omitempty,min=1"`
	Password *string `json:"password,omitempty" validate:"omitempty,min=8"`
	IsActive *bool   `json:"is_active,omitempty"`
	RoleIDs  []int64 `json:"role_ids,omitempty"`
}

// UserUpdater - интерфейс репозитория (использует DTO из storage)
type UserUpdater interface {
	EditUser(ctx context.Context, userID int64, passwordHash []byte, req dto.EditUserRequest) error
	ReplaceUserRoles(ctx context.Context, userID int64, roleIDs []int64) error
}

func New(log *slog.Logger, updater UserUpdater) http.HandlerFunc {
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

		// 2. Декодируем JSON
		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
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

		log.Info("user updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
