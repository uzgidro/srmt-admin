package edit

import (
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// ContactUpdater - interface for updating contacts
type ContactUpdater interface {
	EditContact(ctx context.Context, contactID int64, req dto.EditContactRequest, iconFile *multipart.FileHeader) error
}

func New(log *slog.Logger, updater ContactUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.edit.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req dto.EditContactRequest
		var iconFile *multipart.FileHeader

		if formparser.IsMultipartForm(r) {
			// Parse multipart form
			const maxUploadSize = 10 * 1024 * 1024 // 10 MB for icon
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request or file is too large"))
				return
			}

			// Parse strings
			req.Name = formparser.GetFormString(r, "name")
			req.Email = formparser.GetFormString(r, "email")
			req.Phone = formparser.GetFormString(r, "phone")
			req.IPPhone = formparser.GetFormString(r, "ip_phone")
			req.ExternalOrgName = formparser.GetFormString(r, "external_organization_name")

			// Parse IDs
			var err error
			req.OrganizationID, err = formparser.GetFormInt64(r, "organization_id")
			if err != nil {
				log.Error("invalid organization_id", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
				return
			}

			req.DepartmentID, err = formparser.GetFormInt64(r, "department_id")
			if err != nil {
				log.Error("invalid department_id", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid department_id"))
				return
			}

			req.PositionID, err = formparser.GetFormInt64(r, "position_id")
			if err != nil {
				log.Error("invalid position_id", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid position_id"))
				return
			}

			// Parse Date
			req.DOB, err = formparser.GetFormDate(r, "dob")
			if err != nil {
				log.Error("invalid dob", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid dob format"))
				return
			}

			// Handle icon file upload if present
			iconFile, err = formparser.GetFormFile(r, "icon")
			if err != nil {
				log.Error("failed to get icon file", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}
		} else {
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

		err = updater.EditContact(r.Context(), id, req, iconFile)
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
				render.JSON(w, r, resp.Conflict("Email or phone already exists"))
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
		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
