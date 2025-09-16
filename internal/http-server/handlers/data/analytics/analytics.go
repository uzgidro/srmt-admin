package analytics

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"strconv"
)

func New(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.data.analytics.New"

		// Создаем логгер с контекстом запроса для лучшей трассировки
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Получаем 'id' из URL
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("неверный формат ID", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Неверный формат ID"))
			return
		}

		log.Info("получен запрос на аналитику", slog.Int64("id", id))

		// 2. Здесь будет ваша бизнес-логика
		// Например, вызов метода репозитория для получения данных по 'id'.
		// В качестве примера вернем заглушку.

		// Пример структуры ответа
		analyticsData := struct {
			ID          int64    `json:"id"`
			Metrics     []string `json:"metrics"`
			Description string   `json:"description"`
		}{
			ID:          id,
			Metrics:     []string{"users_online", "requests_per_minute", "error_rate"},
			Description: "Это пример аналитических данных для объекта с указанным ID.",
		}

		// 3. Отправляем успешный JSON-ответ
		render.JSON(w, r, analyticsData)
	}
}
