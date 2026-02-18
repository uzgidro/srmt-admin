package refresh

import (
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Response struct {
	AccessToken string `json:"access_token"`
}

// New создает новый HTTP-хендлер для обновления токенов.
func New(log *slog.Logger, authProvider *auth.Service) http.HandlerFunc {
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

		// Refresh via service
		pair, refreshTTL, err := authProvider.Refresh(r.Context(), refreshToken)
		if err != nil {
			// TODO: handle generic invalid token error vs valid but expired vs server error if needed
			// For now, assuming most errors here mean unauthorized
			if errors.Is(err, storage.ErrUserNotFound) {
				log.Warn("user not found during refresh", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Internal server error"))
				return
			}
			log.Warn("failed to refresh token", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Invalid or expired refresh token"))
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

		log.Info("token refreshed successfully")

		// 3. Отправляем новую пару токенов
		render.JSON(w, r, Response{
			AccessToken: pair.AccessToken,
		})
	}
}
