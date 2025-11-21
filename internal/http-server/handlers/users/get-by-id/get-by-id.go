package get_by_id

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"strconv"
)

type UserGetter interface {
	GetUserByID(ctx context.Context, id int64) (*user.Model, error)
}

func New(log *slog.Logger, getter UserGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.get_by_id.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Получаем ID из URL
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// 2. Вызываем репозиторий
		user, err := getter.GetUserByID(r.Context(), id)
		if err != nil {
			// 3. Обрабатываем ошибку "Не найдено"
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("user not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("User not found"))
				return
			}
			// 4. Обрабатываем остальные ошибки
			log.Error("failed to get user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve user"))
			return
		}

		log.Info("successfully retrieved user", slog.Int64("id", user.ID))
		render.JSON(w, r, user)
	}
}
