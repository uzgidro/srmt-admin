package sign_in

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
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
)

type Request struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type Response struct {
	resp.Response
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type UserGetter interface {
	GetUserByName(ctx context.Context, name string) (user.Model, error)
}

type TokenCreator interface {
	Create(u user.Model) (token.Pair, error)
}

func New(log *slog.Logger, userGetter UserGetter, tokenCreator TokenCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.auth.sign-in.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

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

		// get user
		u, err := userGetter.GetUserByName(r.Context(), req.Name)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) {
				log.Warn("user not found", slog.String("name", req.Name))
				render.JSON(w, r, resp.BadRequest("invalid credentials"))
				return
			}
			log.Error("failed to get user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Internal server error"))
			return
		}

		// check password
		if err := bcrypt.CompareHashAndPassword([]byte(u.PassHash), []byte(req.Password)); err != nil {
			log.Warn("invalid password")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid credentials"))
			return
		}

		pair, err := tokenCreator.Create(u)
		if err != nil {
			log.Error("failed to create pair", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Internal server error"))
			return
		}
		render.JSON(w, r, Response{resp.Ok(), pair.AccessToken, pair.RefreshToken})
		return
	}
}
