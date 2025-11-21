package shutdowns

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/shutdown"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type shutdownGetter interface {
	GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
	GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error)
}

const layout = "2006-01-02" // YYYY-MM-DD

func Get(log *slog.Logger, getter shutdownGetter, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var day time.Time
		dateStr := r.URL.Query().Get("date")

		if dateStr == "" {
			day = time.Now().In(loc)
		} else {
			var err error
			// Parse the date in the configured timezone
			day, err = time.ParseInLocation(layout, dateStr, loc)
			if err != nil {
				log.Warn("invalid 'date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'date' format, use YYYY-MM-DD"))
				return
			}
		}

		shutdowns, err := getter.GetShutdowns(r.Context(), day)
		if err != nil {
			log.Error("failed to get all shutdowns", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve shutdowns"))
			return
		}

		orgTypesMap, err := getter.GetOrganizationTypesMap(r.Context())
		if err != nil {
			log.Error("failed to get organization types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organization types"))
			return
		}

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

			wasGrouped := false
			for _, t := range types {
				switch t {
				case "ges":
					response.Ges = append(response.Ges, s)
					wasGrouped = true
				case "mini":
					response.Mini = append(response.Mini, s)
					wasGrouped = true
				case "micro":
					response.Micro = append(response.Micro, s)
					wasGrouped = true
				}
			}
			if !wasGrouped {
				response.Other = append(response.Other, s)
			}
		}

		log.Info("successfully retrieved and grouped shutdowns", slog.Int("count", len(shutdowns)))
		render.JSON(w, r, response)
	}
}
