package delete

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"srmt-admin/internal/storage"
	"strconv"
)

type RoleDeleter interface {
	DeleteRole(ctx context.Context, id int64) error
}

func New(log *slog.Logger, roleDeleter RoleDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(slog.String("op", "handler.role.delete.New"))

		// Извлекаем ID роли из URL-параметра
		roleID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid role ID", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "invalid role id"})
			return
		}

		err = roleDeleter.DeleteRole(r.Context(), roleID)
		if err != nil {
			// Если роль не найдена, возвращаем 404 Not Found
			if errors.Is(err, storage.ErrRoleNotFound) {
				log.Warn("role not found to delete", "role_id", roleID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, map[string]string{"error": "role not found"})
				return
			}
			// Все остальные ошибки — внутренние
			log.Error("failed to delete role", "error", err)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "internal error"})
			return
		}

		log.Info("role deleted successfully", "role_id", roleID)

		// При успешном удалении принято возвращать 204 No Content.
		render.Status(r, http.StatusNoContent)
	}
}
