package refresh

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/token"
	"time"
)

type Response struct {
	AccessToken string `json:"access_token"`
}

type UserGetter interface {
	GetUserByID(ctx context.Context, id int64) (*user.Model, error)
}

type TokenRefresher interface {
	Verify(token string) (*token.Claims, error)
	Create(u *user.Model) (token.Pair, error)
	GetRefreshTTL() time.Duration
}

// New создает новый HTTP-хендлер для обновления токенов.
func New(log *slog.Logger, userService UserGetter, refresher TokenRefresher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.auth.refresh.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		cookie, err := r.Cookie("refresh_token")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				log.Warn("refresh token cookie not found")
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Unauthorized("Refresh token not provided"))
				return
			}
			// Другие возможные ошибки
			log.Warn("failed to get refresh token cookie", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request"))
			return
		}

		refreshToken := cookie.Value

		claims, err := refresher.Verify(refreshToken)
		if err != nil {
			log.Warn("failed to verify token", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Invalid or expired refresh token"))
			return
		}

		u, err := userService.GetUserByID(r.Context(), claims.UserID)
		if err != nil {
			log.Warn("failed to get user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Internal server error"))
			return
		}

		pair, err := refresher.Create(u)
		if err != nil {
			log.Warn("failed to create pair", sl.Err(err))
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
			MaxAge:      int(refresher.GetRefreshTTL()), // Время жизни cookie
		})

		log.Info("token refreshed successfully")

		// 3. Отправляем новую пару токенов
		render.JSON(w, r, Response{
			AccessToken: pair.AccessToken,
		})
	}
}
