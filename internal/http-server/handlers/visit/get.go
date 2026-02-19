package visit

import (
	"context"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/visit"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type visitGetter interface {
	GetVisits(ctx context.Context, day time.Time) ([]*visit.ResponseModel, error)
}

const dateLayout = "2006-01-02"

func Get(log *slog.Logger, getter visitGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.visit.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var day time.Time

		// Parse date using formparser with location support
		dateVal, err := formparser.GetFormDateInLocation(r, "date", loc)
		if err != nil {
			log.Warn("invalid 'date' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'date' format, use YYYY-MM-DD"))
			return
		}

		if dateVal == nil {
			now := time.Now().In(loc)
			// День начинается в 07:00 местного времени
			day = time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, loc)
			log.Info("no 'date' parameter provided, using today", "date", day.Format(dateLayout))
		} else {
			// День начинается в 07:00 местного времени
			t := *dateVal
			day = time.Date(t.Year(), t.Month(), t.Day(), 7, 0, 0, 0, loc)
			log.Info("using provided 'date' parameter", "date", t.Format(dateLayout))
		}

		visits, err := getter.GetVisits(r.Context(), day)
		if err != nil {
			log.Error("failed to get visits", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve visits"))
			return
		}

		// Transform incidents to include presigned URLs
		visitsWithURLs := make([]*visit.ResponseWithURLs, 0, len(visits))
		for _, inc := range visits {
			incWithURLs := &visit.ResponseWithURLs{
				ID:               inc.ID,
				ResponsibleName:  inc.ResponsibleName,
				VisitDate:        inc.VisitDate,
				Description:      inc.Description,
				CreatedAt:        inc.CreatedAt,
				OrganizationID:   inc.OrganizationID,
				OrganizationName: inc.OrganizationName,
				CreatedByUser:    inc.CreatedByUser,
				Files:            helpers.TransformFilesWithURLs(r.Context(), inc.Files, minioRepo, log),
			}
			visitsWithURLs = append(visitsWithURLs, incWithURLs)
		}

		log.Info("successfully retrieved visits", slog.Int("count", len(visits)))

		render.JSON(w, r, visitsWithURLs)
	}
}
