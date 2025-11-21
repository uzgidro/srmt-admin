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

// DepartmentDeleter - интерфейс для удаления
type DepartmentDeleter interface {
	DeleteDepartment(ctx context.Context, id int64) error
}

func New(log *slog.Logger, deleter DepartmentDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.department.delete.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Получаем ID из URL
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter, must be a number"))
			return
		}

		// 2. Вызываем метод репозитория
		err = deleter.DeleteDepartment(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("department not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Department not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("department has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete department: it has associated contacts"))
				return
			}
			log.Error("failed to delete department", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete department"))
			return
		}

		log.Info("department deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
