package refresh

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/token"
)

type Request struct {
	RefreshToken string `json:"refresh_token"`
}

// Response определяет структуру для успешного ответа.
type Response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserGetter interface {
	GetUserByID(ctx context.Context, id int64) (user.Model, error)
}

type TokenRefresher interface {
	Verify(token string) (*token.Claims, error)
	Create(u user.Model) (token.Pair, error)
}

// New создает новый HTTP-хендлер для обновления токенов.
func New(log *slog.Logger, userService UserGetter, refresher TokenRefresher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.auth.refresh.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Декодируем JSON из тела запроса
		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if req.RefreshToken == "" {
			log.Warn("refresh token is empty")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Refresh token is required"))
			return
		}

		claims, err := refresher.Verify(req.RefreshToken)
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

		log.Info("token refreshed successfully")

		// 3. Отправляем новую пару токенов
		render.JSON(w, r, Response{
			AccessToken:  pair.AccessToken,
			RefreshToken: pair.RefreshToken,
		})
	}
}
