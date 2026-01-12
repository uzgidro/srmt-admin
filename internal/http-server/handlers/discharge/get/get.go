package get

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/discharge"
	"strconv"
	"time"
)

type DischargeGetter interface {
	GetDischargesByCascades(ctx context.Context, isOngoing *bool, startDate, endDate *time.Time) ([]discharge.Cascade, error)
}

func New(log *slog.Logger, getter DischargeGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
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
			t, err := time.ParseInLocation(layout, startDateStr, loc)
			if err != nil {
				log.Warn("invalid 'start_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'start_date' format, use YYYY-MM-DD"))
				return
			}
			// День начинается в 07:00 местного времени
			t = time.Date(t.Year(), t.Month(), t.Day(), 7, 0, 0, 0, loc)
			startDate = &t
		}

		if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
			t, err := time.ParseInLocation(layout, endDateStr, loc)
			if err != nil {
				log.Warn("invalid 'end_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'end_date' format, use YYYY-MM-DD"))
				return
			}
			// День заканчивается в 07:00 местного времени следующего дня
			t = time.Date(t.Year(), t.Month(), t.Day(), 7, 0, 0, 0, loc).Add(24 * time.Hour)
			endDate = &t
		}

		// 2. Вызываем метод репозитория с фильтрами
		cascades, err := getter.GetDischargesByCascades(r.Context(), isOngoing, startDate, endDate)
		if err != nil {
			log.Error("failed to get all discharges", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve discharges"))
			return
		}

		// 3. Transform cascades to include presigned URLs for all nested discharges
		cascadesWithURLs := make([]discharge.CascadeWithURLs, 0, len(cascades))
		for _, c := range cascades {
			hppsWithURLs := make([]discharge.HPPWithURLs, 0, len(c.HPPs))
			for _, hpp := range c.HPPs {
				dischargesWithURLs := make([]discharge.ModelWithURLs, 0, len(hpp.Discharges))
				for _, d := range hpp.Discharges {
					dWithURLs := discharge.ModelWithURLs{
						ID:             d.ID,
						Organization:   d.Organization,
						CreatedByUser:  d.CreatedByUser,
						ApprovedByUser: d.ApprovedByUser,
						StartedAt:      d.StartedAt,
						EndedAt:        d.EndedAt,
						FlowRate:       d.FlowRate,
						TotalVolume:    d.TotalVolume,
						Reason:         d.Reason,
						IsOngoing:      d.IsOngoing,
						Approved:       d.Approved,
						Files:          helpers.TransformFilesWithURLs(r.Context(), d.Files, minioRepo, log),
					}
					dischargesWithURLs = append(dischargesWithURLs, dWithURLs)
				}
				hppWithURLs := discharge.HPPWithURLs{
					ID:          hpp.ID,
					Name:        hpp.Name,
					TotalVolume: hpp.TotalVolume,
					Discharges:  dischargesWithURLs,
				}
				hppsWithURLs = append(hppsWithURLs, hppWithURLs)
			}
			cascadeWithURLs := discharge.CascadeWithURLs{
				ID:          c.ID,
				Name:        c.Name,
				TotalVolume: c.TotalVolume,
				HPPs:        hppsWithURLs,
			}
			cascadesWithURLs = append(cascadesWithURLs, cascadeWithURLs)
		}

		log.Info("successfully retrieved discharges", slog.Int("count", len(cascadesWithURLs)))
		render.JSON(w, r, cascadesWithURLs)
	}
}
