package me

import (
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// Response определяет структуру для ответа, содержащего информацию о пользователе.
type Response struct {
	ID    int64    `json:"id"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// New создает новый HTTP-хендлер для эндпоинта /auth/me.
func New(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.auth.me.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Извлекаем claims из контекста, куда их положил middleware.
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			// Эта ситуация не должна происходить, если middleware настроен правильно.
			// Это внутренняя ошибка сервера.
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Internal server error"))
			return
		}

		log.Info("user data retrieved successfully", slog.Int64("user_id", claims.UserID))

		// 2. Формируем и отправляем ответ.
		render.JSON(w, r, Response{
			ID:    claims.UserID,
			Name:  claims.Name,
			Roles: claims.Roles,
		})
	}
}
