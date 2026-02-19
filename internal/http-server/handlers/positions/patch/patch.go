package patch

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

// PositionUpdater - интерфейс для обновления
type PositionUpdater interface {
	EditPosition(ctx context.Context, id int64, req dto.EditPositionRequest) error
}

func New(log *slog.Logger, updater PositionUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.position.update.New"
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
		var req dto.EditPositionRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// 3. Валидация
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// 4. Вызываем метод сервиса
		err = updater.EditPosition(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("position not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Position not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("position name duplicate")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Position with this name already exists"))
				return
			}
			log.Error("failed to update position", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update position"))
			return
		}

		log.Info("position updated", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
