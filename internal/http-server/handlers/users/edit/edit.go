package edit

import (
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"strconv"
	"strings"

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

		contentType := r.Header.Get("Content-Type")
		isMultipart := strings.Contains(contentType, "multipart/form-data")

		var req dto.UpdateUserRequest
		var iconFile *multipart.FileHeader

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
				var roles []int64
				rolesStrs := strings.Split(rolesStr, ",")
				for _, roleStr := range rolesStrs {
					roleID, err := strconv.ParseInt(strings.TrimSpace(roleStr), 10, 64)
					if err != nil {
						log.Error("invalid role_id", sl.Err(err), "value", roleStr)
						render.Status(r, http.StatusBadRequest)
						render.JSON(w, r, resp.BadRequest("Invalid role_ids format"))
						return
					}
					roles = append(roles, roleID)
				}
				req.RoleIDs = &roles
			}

			// Get icon file
			// We don't process it here, just pass the header to service
			var err error
			_, iconFile, err = r.FormFile("icon")
			if err != nil {
				if err != http.ErrMissingFile {
					log.Error("failed to get icon file", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid icon file"))
					return
				}
				iconFile = nil // No file
			}

		} else {
			// Parse JSON
			// Temporary struct to match JSON payload (role_ids omitempty)
			type jsonRequest struct {
				Login    *string `json:"login,omitempty"`
				Password *string `json:"password,omitempty"`
				IsActive *bool   `json:"is_active,omitempty"`
				RoleIDs  []int64 `json:"role_ids,omitempty"`
			}
			var jReq jsonRequest
			if err := render.DecodeJSON(r.Body, &jReq); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			req.Login = jReq.Login
			req.Password = jReq.Password
			req.IsActive = jReq.IsActive
			// Check if RoleIDs was present (len 0 can mean clear, but nil means no change)
			// Wait, jsonRequest.RoleIDs is a slice. If key missing -> nil. If key present "role_ids": [] -> empty slice.
			// Perfect, using pointer in DTO.
			if jReq.RoleIDs != nil {
				req.RoleIDs = &jReq.RoleIDs
			}
		}

		// Validation (Basic constraints can be checked here or in service, but handlers usually validate format)
		// DTO `UpdateUserRequest` doesn't have tags? Use `Request` struct if needed for validation tags.
		// For now, minimal validation logic as fields are pointers (optional).
		// If we want validation tags, we should have used a dedicated Request struct or tagged DTO.
		// Existing code used `Request` struct with tags.
		// Let's assume validation is minimal or service handles business validation.
		// The `min=8` for password was in handler.
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
				render.JSON(w, r, resp.BadRequest("Login already exists"))
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
