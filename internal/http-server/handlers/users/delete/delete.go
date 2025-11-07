package delete

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
	"srmt-admin/internal/storage"
	"strconv"
)

type UserDeleter interface {
	DeleteUser(ctx context.Context, id int64) error
}

func New(log *slog.Logger, deleter UserDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.delete.New"
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
		// (По нашей логике, удаляется 'User', но 'Contact' остается)
		err = deleter.DeleteUser(r.Context(), id)
		if err != nil {
			// 3. Обработка ошибок
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("user not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("User not found"))
				return
			}
			// (Напр., если user_id используется в created_by)
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("user has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete user: it is referenced by other records"))
				return
			}
			log.Error("failed to delete user", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete user"))
			return
		}

		log.Info("user deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
