package edit

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"
)

type Request struct {
	Name     *string `json:"name,omitempty"`
	Password *string `json:"password,omitempty" validate:"omitempty,dive,min=8"`
}

type Response struct {
	resp.Response
}

type UserEditor interface {
	EditUser(ctx context.Context, id int64, name, passHash string) error
}

func New(log *slog.Logger, editor UserEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.edit.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
		if err != nil {
			log.Warn("invalid user ID", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid user id"))
			return
		}

		var req Request

		// Decode JSON
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		log.Info("request parsed", slog.Any("req", req))

		// Validate fields
		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)

			log.Error("failed to validate request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to validate request"))

			return
		}

		var newName, newPassHash string
		if req.Name != nil {
			newName = *req.Name
		}
		if req.Password != nil {
			hashedPassword, passErr := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
			if passErr != nil {
				log.Error("failed to hash password", sl.Err(passErr))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("internal server error"))
				return
			}
			newPassHash = string(hashedPassword)
		}

		err = editor.EditUser(r.Context(), userID, newName, newPassHash)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				log.Warn("user not found to edit", "user_id", userID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("user not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("username already exists", "name", newName)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.BadRequest("username already exists"))
				return
			}

			log.Error("failed to edit user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to edit user"))
			return
		}

		log.Info("user successfully edited", slog.Int64("id", userID))

		render.Status(r, http.StatusOK)
	}
}
