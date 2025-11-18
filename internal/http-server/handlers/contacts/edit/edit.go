package edit

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"
	"time"
)

// Request (DTO хендлера)
type Request struct {
	Name            *string    `json:"name,omitempty" validate:"omitempty,min=1"`
	Email           *string    `json:"email,omitempty" validate:"omitempty,email"`
	Phone           *string    `json:"phone,omitempty"`
	IPPhone         *string    `json:"ip_phone,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	ExternalOrgName *string    `json:"external_organization_name,omitempty"`
	OrganizationID  *int64     `json:"organization_id,omitempty"`
	DepartmentID    *int64     `json:"department_id,omitempty"`
	PositionID      *int64     `json:"position_id,omitempty"`
}

// ContactUpdater - интерфейс репозитория
type ContactUpdater interface {
	EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest) error
}

func New(log *slog.Logger, updater ContactUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.update.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

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
		storageReq := dto.EditContactRequest{
			Name:            req.Name,
			Email:           req.Email,
			Phone:           req.Phone,
			IPPhone:         req.IPPhone,
			DOB:             req.DOB,
			ExternalOrgName: req.ExternalOrgName,
			OrganizationID:  req.OrganizationID,
			DepartmentID:    req.DepartmentID,
			PositionID:      req.PositionID,
		}

		err = updater.EditContact(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("contact not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Contact not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("duplicate data on update")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.BadRequest("Email or phone already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization, department or position"))
				return
			}
			log.Error("failed to update contact", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update contact"))
			return
		}

		log.Info("contact updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
