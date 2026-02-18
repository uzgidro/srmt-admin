package sign_in

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Name     string `json:"name" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type Response struct {
	resp.Response
	AccessToken string `json:"access_token,omitempty"`
}

type AuthProvider interface {
	Login(ctx context.Context, login, password string) (token.Pair, time.Duration, error)
}

func New(log *slog.Logger, authProvider *auth.Service) http.HandlerFunc {
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

		// Login via service
		pair, refreshTTL, err := authProvider.Login(r.Context(), req.Name, req.Password)
		if err != nil {
			if errors.Is(err, storage.ErrUserNotFound) || errors.Is(err, storage.ErrInvalidCredentials) {
				log.Warn("invalid credentials", slog.String("name", req.Name))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("invalid credentials"))
				return
			}
			if errors.Is(err, storage.ErrUserDeactivated) {
				log.Warn("user is not active", slog.String("name", req.Name))
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("user account is not active"))
				return
			}
			log.Error("failed to login", sl.Err(err))
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
			MaxAge:      int(refreshTTL.Seconds()), // Время жизни cookie
		})

		render.JSON(w, r, Response{resp.OK(), pair.AccessToken})
	}
}
