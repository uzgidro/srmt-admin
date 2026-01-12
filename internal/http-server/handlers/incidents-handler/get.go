package incidents_handler

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/incident" // (Импорт ResponseModel)
	"time"
)

type incidentGetter interface {
	GetIncidents(ctx context.Context, day time.Time) ([]*incident.ResponseModel, error)
}

const layout = "2006-01-02"

func Get(log *slog.Logger, getter incidentGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.incident.get_all.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var day time.Time
		dateStr := r.URL.Query().Get("date")

		if dateStr == "" {
			now := time.Now().In(loc)
			// День начинается в 07:00 местного времени
			day = time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, loc)
			log.Info("no 'date' parameter provided, using today", "date", day.Format(layout))
		} else {
			var err error
			// Parse the date in the configured timezone
			t, err := time.ParseInLocation(layout, dateStr, loc)
			if err != nil {
				log.Warn("invalid 'date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'date' format, use YYYY-MM-DD"))
				return
			}
			// День начинается в 07:00 местного времени
			day = time.Date(t.Year(), t.Month(), t.Day(), 7, 0, 0, 0, loc)
			log.Info("using provided 'date' parameter", "date", dateStr)
		}

		incidents, err := getter.GetIncidents(r.Context(), day)
		if err != nil {
			log.Error("failed to get all incidents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve incidents"))
			return
		}

		// Transform incidents to include presigned URLs
		incidentsWithURLs := make([]*incident.ResponseWithURLs, 0, len(incidents))
		for _, inc := range incidents {
			incWithURLs := &incident.ResponseWithURLs{
				ID:               inc.ID,
				IncidentTime:     inc.IncidentTime,
				Description:      inc.Description,
				CreatedAt:        inc.CreatedAt,
				OrganizationID:   inc.OrganizationID,
				OrganizationName: inc.OrganizationName,
				CreatedByUser:    inc.CreatedByUser,
				Files:            helpers.TransformFilesWithURLs(r.Context(), inc.Files, minioRepo, log),
			}
			incidentsWithURLs = append(incidentsWithURLs, incWithURLs)
		}

		log.Info("successfully retrieved incidents", slog.Int("count", len(incidentsWithURLs)))

		render.JSON(w, r, incidentsWithURLs)
	}
}
