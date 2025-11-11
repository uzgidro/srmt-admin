package get

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/discharge"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type DischargeGetter interface {
	GetAllDischarges(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Model, error)
}

func New(log *slog.Logger, getter DischargeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Парсим фильтры из query-параметров
		var isOngoing *bool
		if isOngoingStr := r.URL.Query().Get("is_ongoing"); isOngoingStr != "" {
			val, err := strconv.ParseBool(isOngoingStr)
			if err != nil {
				log.Warn("invalid 'is_ongoing' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'is_ongoing' parameter, must be 'true' or 'false'"))
				return
			}
			isOngoing = &val
		}

		var startDate, endDate *time.Time
		const layout = "2006-01-02" // Формат даты YYYY-MM-DD

		if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
			t, err := time.Parse(layout, startDateStr)
			if err != nil {
				log.Warn("invalid 'start_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'start_date' format, use YYYY-MM-DD"))
				return
			}
			startDate = &t
		}

		if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
			t, err := time.Parse(layout, endDateStr)
			if err != nil {
				log.Warn("invalid 'end_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'end_date' format, use YYYY-MM-DD"))
				return
			}
			// Чтобы включить весь день, устанавливаем время на конец дня
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endDate = &t
		}

		// 2. Вызываем метод репозитория с фильтрами
		discharges, err := getter.GetAllDischarges(r.Context(), isOngoing, startDate, endDate)
		if err != nil {
			log.Error("failed to get all discharges", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve discharges"))
			return
		}

		log.Info("successfully retrieved discharges", slog.Int("count", len(discharges)))
		render.JSON(w, r, discharges)
	}
}
