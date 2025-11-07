package add

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"time"
)

type newContactRequest struct {
	FIO             string     `json:"fio" validate:"required"`
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
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`

	// XOR: Либо `contact_id`, либо `contact`
	ContactID *int64             `json:"contact_id,omitempty" validate:"omitempty,gt=0"`
	Contact   *newContactRequest `json:"contact,omitempty" validate:"omitempty,dive"`
}

type Response struct {
	resp.Response
	ID int64 `json:"id"` // ID нового пользователя (из users)
}

// UserLinker - интерфейс для репозитория Users
type UserLinker interface {
	AddUser(ctx context.Context, login string, passwordHash []byte, contactID int64) (int64, error)
	IsContactLinked(ctx context.Context, contactID int64) (bool, error)
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
}

// New - Фабрика хендлера
// (Мы внедряем *два* репозитория/интерфейса)
func New(log *slog.Logger, userRepo UserLinker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
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
				FIO:             req.Contact.FIO,
				Email:           req.Contact.Email,
				Phone:           req.Contact.Phone,
				IPPhone:         req.Contact.IPPhone,
				DOB:             req.Contact.DOB,
				ExternalOrgName: req.Contact.ExternalOrgName,
				OrganizationID:  req.Contact.OrganizationID,
				DepartmentID:    req.Contact.DepartmentID,
				PositionID:      req.Contact.PositionID,
			}
			newContactID, err := userRepo.AddContact(r.Context(), storageReq)
			if err != nil {
				// (Обрабатываем ошибки от AddContact)
				if errors.Is(err, storage.ErrDuplicate) {
					log.Warn("duplicate contact data", "fio", req.Contact.FIO)
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

		log.Info("user added", slog.Int64("id", newUserID))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: newUserID})
	}
}
