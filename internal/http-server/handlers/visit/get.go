package visit

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
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

func Get(log *slog.Logger, getter visitGetter, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.visit.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var day time.Time
		dateStr := r.URL.Query().Get("date")

		if dateStr == "" {
			day = time.Now().In(loc)
			log.Info("no 'date' parameter provided, using today", "date", day.Format(dateLayout))
		} else {
			var err error
			// Parse the date in the configured timezone
			day, err = time.ParseInLocation(dateLayout, dateStr, loc)
			if err != nil {
				log.Warn("invalid 'date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'date' format, use YYYY-MM-DD"))
				return
			}
			log.Info("using provided 'date' parameter", "date", dateStr)
		}

		visits, err := getter.GetVisits(r.Context(), day)
		if err != nil {
			log.Error("failed to get visits", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve visits"))
			return
		}

		log.Info("successfully retrieved visits", slog.Int("count", len(visits)))

		render.JSON(w, r, visits)
	}
}
