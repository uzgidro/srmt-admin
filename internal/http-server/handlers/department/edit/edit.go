package edit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// DepartmentEditor - интерфейс для обновления
type DepartmentEditor interface {
	EditDepartment(ctx context.Context, id int64, req dto.EditDepartmentRequest) error
}

func New(log *slog.Logger, updater DepartmentEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.department.update.New"
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

		// 2. Декодируем JSON
		var req dto.EditDepartmentRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// 3. Валидация
		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(validationErrors))
			return
		}

		// 4. Вызываем метод сервиса
		err = updater.EditDepartment(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("department not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Department not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("department name duplicate")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Department with this name already exists"))
				return
			}
			log.Error("failed to update department", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update department"))
			return
		}

		log.Info("department updated", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
