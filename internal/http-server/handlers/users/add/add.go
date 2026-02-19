package add

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"

	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

type UserCreator interface {
	CreateUser(ctx context.Context, req dto.CreateUserRequest, iconFile *multipart.FileHeader) (int64, error)
}

func New(log *slog.Logger, userCreator UserCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req dto.CreateUserRequest
		var iconFile *multipart.FileHeader
		var err error

		if formparser.IsMultipartForm(r) {
			// 10 MB limit for icon
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request or file is too large"))
				return
			}

			req, err = parseMultipartRequest(r)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

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

		// Validation
		if err := validator.New().Struct(req); err != nil {
			// Additional XOR check for ContactID vs Contact
			if (req.ContactID == nil && req.Contact == nil) || (req.ContactID != nil && req.Contact != nil) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Must provide either 'contact_id' or 'contact' object, but not both"))
				return
			}

			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(validationErrors))
			return
		}

		// Double check XOR logic manually if validator didn't catch it logic-wise (validator handles presence, but not XOR logic perfectly with simple tags sometimes)
		// `dto.CreateUserRequest` doesn't have `oneof` tag on these fields easily because one is int64, other is struct pointer.
		if (req.ContactID == nil && req.Contact == nil) || (req.ContactID != nil && req.Contact != nil) {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Must provide either 'contact_id' or 'contact' object, but not both"))
			return
		}

		newUserID, err := userCreator.CreateUser(r.Context(), req, iconFile)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "already exists") {
				log.Warn("duplicate/conflict", sl.Err(err))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict(err.Error()))
				return
			}
			if strings.Contains(err.Error(), "validation failed") || strings.Contains(err.Error(), "not found") {
				log.Warn("bad request error", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			log.Error("failed to create user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create user"))
			return
		}

		log.Info("user added", slog.Int64("id", newUserID))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: newUserID})
	}
}

func parseMultipartRequest(r *http.Request) (dto.CreateUserRequest, error) {
	var req dto.CreateUserRequest
	var err error

	req.Login, err = formparser.GetFormStringRequired(r, "login")
	if err != nil {
		return req, err
	}

	req.Password, err = formparser.GetFormStringRequired(r, "password")
	if err != nil {
		return req, err
	}

	req.RoleIDs, err = formparser.GetFormInt64Slice(r, "role_ids")
	if err != nil {
		return req, fmt.Errorf("invalid role_ids: %w", err)
	}
	if len(req.RoleIDs) == 0 {
		return req, errors.New("field role_ids is required")
	}

	// Contact ID
	req.ContactID, err = formparser.GetFormInt64(r, "contact_id")
	if err != nil {
		return req, err
	}

	// Parse contact object if present
	if contactName := r.FormValue("contact.name"); contactName != "" {
		contact := &dto.AddContactRequest{
			Name:            contactName,
			Email:           formparser.GetFormString(r, "contact.email"),
			Phone:           formparser.GetFormString(r, "contact.phone"),
			IPPhone:         formparser.GetFormString(r, "contact.ip_phone"),
			ExternalOrgName: formparser.GetFormString(r, "contact.external_organization_name"),
		}

		if contact.DOB, err = formparser.GetFormDate(r, "contact.dob"); err != nil {
			return req, fmt.Errorf("invalid contact.dob: %w", err)
		}

		if contact.OrganizationID, err = formparser.GetFormInt64(r, "contact.organization_id"); err != nil {
			return req, err
		}
		if contact.DepartmentID, err = formparser.GetFormInt64(r, "contact.department_id"); err != nil {
			return req, err
		}
		if contact.PositionID, err = formparser.GetFormInt64(r, "contact.position_id"); err != nil {
			return req, err
		}

		req.Contact = contact
	}

	return req, nil
}
