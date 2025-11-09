package shutdowns

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/shutdown" // (Импорт ResponseModel и GroupedResponse)
	"time"
)

type shutdownGetter interface {
	GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
	GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error)
}

const layout = "2006-01-02" // YYYY-MM-DD

func Get(log *slog.Logger, getter shutdownGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Парсим 'date' (как в Incidents)
		var day time.Time
		dateStr := r.URL.Query().Get("date")

		if dateStr == "" {
			day = time.Now()
		} else {
			var err error
			day, err = time.Parse(layout, dateStr)
			if err != nil {
				log.Warn("invalid 'date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'date' format, use YYYY-MM-DD"))
				return
			}
		}

		// 2. Вызываем репозиторий (получаем плоский список)
		shutdowns, err := getter.GetShutdowns(r.Context(), day)
		if err != nil {
			log.Error("failed to get all shutdowns", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve shutdowns"))
			return
		}

		// 3. (НОВАЯ ЛОГИКА) Получаем типы организаций
		orgTypesMap, err := getter.GetOrganizationTypesMap(r.Context())
		if err != nil {
			log.Error("failed to get organization types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organization types"))
			return
		}

		// 4. (НОВАЯ ЛОГИКА) Группируем по типу
		response := shutdown.GroupedResponse{
			Ges:   make([]*shutdown.ResponseModel, 0),
			Mini:  make([]*shutdown.ResponseModel, 0),
			Micro: make([]*shutdown.ResponseModel, 0),
			Other: make([]*shutdown.ResponseModel, 0),
		}

		for _, s := range shutdowns {
			types, ok := orgTypesMap[s.OrganizationID]
			if !ok {
				response.Other = append(response.Other, s)
				continue
			}

			// (Логика может быть сложной, если у ГЭС >1 типа,
			// пока просто ищем первое совпадение)
			found := false
			for _, t := range types {
				switch t { // (Используй тут свои реальные типы)
				case "ГЭС":
					response.Ges = append(response.Ges, s)
					found = true
				case "МиниГЭС":
					response.Mini = append(response.Mini, s)
					found = true
				case "МикроГЭС":
					response.Micro = append(response.Micro, s)
					found = true
				}
				if found {
					break
				}
			}
			if !found {
				response.Other = append(response.Other, s)
			}
		}

		log.Info("successfully retrieved and grouped shutdowns", slog.Int("count", len(shutdowns)))
		render.JSON(w, r, response)
	}
}
