package edit

import (
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// UserUpdater - Interface for updating users
type UserUpdater interface {
	UpdateUser(ctx context.Context, userID int64, req dto.UpdateUserRequest, iconFile *multipart.FileHeader) error
}

func New(log *slog.Logger, updater UserUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.update.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Get ID from URL
		idStr := chi.URLParam(r, "userID")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'userID' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'userID' parameter"))
			return
		}

		var req dto.UpdateUserRequest
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

			// Parse form fields
			req.Login = formparser.GetFormString(r, "login")
			req.Password = formparser.GetFormString(r, "password")

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
			if formparser.HasFormField(r, "role_ids") {
				roles, err := formparser.GetFormInt64Slice(r, "role_ids")
				if err != nil {
					log.Error("invalid role_ids", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid role_ids format"))
					return
				}
				req.RoleIDs = &roles
			}

			// Get icon file
			iconFile, err = formparser.GetFormFile(r, "icon")
			if err != nil {
				log.Error("failed to get icon file", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}
		} else {
			// Parse JSON
			// Reuse DTO directly? dto.UpdateUserRequest matches JSON structure exactly?
			// `type UpdateUserRequest struct { Login *string ... RoleIDs *[]int64 ... }`
			// This matches JSON structure IF `role_ids` is passed as array.
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}
		}

		// Validation
		if req.Password != nil && len(*req.Password) < 8 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Password must be at least 8 characters"))
			return
		}
		if req.Login != nil && len(*req.Login) < 1 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Login cannot be empty"))
			return
		}

		// Call Service
		err = updater.UpdateUser(r.Context(), id, req, iconFile)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Warn("user not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("User not found"))
				return
			}
			if strings.Contains(err.Error(), "duplicate") {
				log.Warn("duplicate login", slog.Int64("id", id))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Login already exists"))
				return
			}
			if strings.Contains(err.Error(), "invalid role_id") {
				log.Warn("invalid role", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("One or more role IDs are invalid"))
				return
			}
			log.Error("failed to update user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update user"))
			return
		}

		log.Info("user updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
