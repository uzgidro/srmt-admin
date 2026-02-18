package add

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

// ContactAdder - Interface for adding contacts
type ContactAdder interface {
	AddContact(ctx context.Context, req dto.AddContactRequest, iconFile *multipart.FileHeader) (int64, error)
}

func New(log *slog.Logger, adder ContactAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req dto.AddContactRequest
		var iconFile *multipart.FileHeader

		if formparser.IsMultipartForm(r) {
			// Parse multipart form
			const maxUploadSize = 10 * 1024 * 1024 // 10 MB for icon
			r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request or file is too large"))
				return
			}

			// Parse form fields using formparser
			parsedReq, err := parseMultipartRequest(r)
			if err != nil {
				log.Error("failed to parse form data", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}
			req = parsedReq

			// Handle icon file upload if present
			_, iconFile, err = r.FormFile("icon")
			if err != nil && !errors.Is(err, http.ErrMissingFile) {
				log.Error("failed to get icon file", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid icon file"))
				return
			}
			// If ErrMissingFile, iconFile remains nil, which is fine

		} else {
			// Parse JSON directly into DTO
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}
		}

		// Validation
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := adder.AddContact(r.Context(), req, iconFile)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicate) || strings.Contains(err.Error(), "duplicate") {
				log.Warn("duplicate contact data", "name", req.Name)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.BadRequest("Contact with this email or phone already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) || strings.Contains(err.Error(), "foreign key") {
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

func parseMultipartRequest(r *http.Request) (dto.AddContactRequest, error) {
	name, err := formparser.GetFormStringRequired(r, "name")
	if err != nil {
		return dto.AddContactRequest{}, err
	}

	req := dto.AddContactRequest{
		Name:            name,
		Email:           formparser.GetFormString(r, "email"),
		Phone:           formparser.GetFormString(r, "phone"),
		IPPhone:         formparser.GetFormString(r, "ip_phone"),
		ExternalOrgName: formparser.GetFormString(r, "external_organization_name"),
	}

	// Parse optional fields
	if dob, err := formparser.GetFormTime(r, "dob", time.DateOnly); err == nil {
		req.DOB = dob
	} else if err != nil && r.FormValue("dob") != "" {
		// Only return error if field IS present but invalid. GetFormTime checks this.
		return dto.AddContactRequest{}, fmt.Errorf("dob: %w", err)
	}

	if orgID, err := formparser.GetFormInt64(r, "organization_id"); err == nil {
		req.OrganizationID = orgID
	} else {
		return dto.AddContactRequest{}, err
	}

	if deptID, err := formparser.GetFormInt64(r, "department_id"); err == nil {
		req.DepartmentID = deptID
	} else {
		return dto.AddContactRequest{}, err
	}

	if posID, err := formparser.GetFormInt64(r, "position_id"); err == nil {
		req.PositionID = posID
	} else {
		return dto.AddContactRequest{}, err
	}

	return req, nil
}
