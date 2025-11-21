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
	"srmt-admin/internal/lib/model/department"
	"srmt-admin/internal/storage"
	"strconv"
)

// DepartmentGetter - интерфейс для получения одного
type DepartmentGetter interface {
	GetDepartmentByID(ctx context.Context, id int64) (*department.Model, error)
}

func New(log *slog.Logger, getter DepartmentGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.department.get_by_id.New"
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
		dep, err := getter.GetDepartmentByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("department not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Department not found"))
				return
			}
			log.Error("failed to get department", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve department"))
			return
		}

		log.Info("successfully retrieved department", slog.Int64("id", dep.ID))
		render.JSON(w, r, dep)
	}
}
