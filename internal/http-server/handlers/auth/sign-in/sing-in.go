package sign_in

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type Response struct {
	resp.Response
	AccessToken string `json:"access_token,omitempty"`
}

type UserGetter interface {
	GetUserByLogin(ctx context.Context, login string) (*user.Model, string, error)
}

type TokenCreator interface {
	Create(u *user.Model) (token.Pair, error)
	GetRefreshTTL() time.Duration
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
		u, pass, err := userGetter.GetUserByLogin(r.Context(), req.Name)
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
		if err := bcrypt.CompareHashAndPassword([]byte(pass), []byte(req.Password)); err != nil {
			log.Warn("invalid password")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid credentials"))
			return
		}

		// check if user is active
		if !u.IsActive {
			log.Warn("user is not active", slog.String("name", req.Name))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("user account is not active"))
			return
		}

		pair, err := tokenCreator.Create(u)
		if err != nil {
			log.Error("failed to create pair", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Internal server error"))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:        "refresh_token",
			Value:       pair.RefreshToken,
			Path:        "/",                   // Доступен на всем сайте
			HttpOnly:    true,                  // Запрещаем доступ из JS
			Secure:      true,                  // Отправлять только по HTTPS (в продакшене)
			SameSite:    http.SameSiteNoneMode, // Защита от CSRF
			Partitioned: true,
			MaxAge:      int(tokenCreator.GetRefreshTTL()), // Время жизни cookie
		})

		render.JSON(w, r, Response{resp.OK(), pair.AccessToken})
		return
	}
}
