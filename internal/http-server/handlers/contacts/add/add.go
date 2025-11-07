package add

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"time"
)

// Request - JSON DTO хендлера
type Request struct {
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

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

// ContactAdder - интерфейс репозитория
type ContactAdder interface {
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
}

func New(log *slog.Logger, adder ContactAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.add.New"
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

		// Маппинг в DTO хранилища
		storageReq := dto.AddContactRequest{
			FIO:             req.FIO,
			Email:           req.Email,
			Phone:           req.Phone,
			IPPhone:         req.IPPhone,
			DOB:             req.DOB,
			ExternalOrgName: req.ExternalOrgName,
			OrganizationID:  req.OrganizationID,
			DepartmentID:    req.DepartmentID,
			PositionID:      req.PositionID,
		}

		id, err := adder.AddContact(r.Context(), storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("duplicate contact data", "fio", req.FIO)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.BadRequest("Contact with this email or phone already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization, department or position"))
				return
			}
			log.Error("failed to add contact", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add contact"))
			return
		}

		log.Info("contact added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: id})
	}
}
