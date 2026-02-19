package shutdowns

import (
	"context"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
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

func Get(log *slog.Logger, getter shutdownGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Get"
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
		} else {
			// День начинается в 07:00 местного времени
			t := *dateVal
			day = time.Date(t.Year(), t.Month(), t.Day(), 7, 0, 0, 0, loc)
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

		// Transform function to convert shutdown to WithURLs version
		transformShutdown := func(s *shutdown.ResponseModel) *shutdown.ResponseWithURLs {
			return &shutdown.ResponseWithURLs{
				ID:                            s.ID,
				OrganizationID:                s.OrganizationID,
				OrganizationName:              s.OrganizationName,
				StartedAt:                     s.StartedAt,
				EndedAt:                       s.EndedAt,
				Reason:                        s.Reason,
				CreatedByUser:                 s.CreatedByUser,
				GenerationLossMwh:             s.GenerationLossMwh,
				CreatedAt:                     s.CreatedAt,
				Viewed:                        s.Viewed,
				IdleDischargeVolumeThousandM3: s.IdleDischargeVolumeThousandM3,
				Files:                         helpers.TransformFilesWithURLs(r.Context(), s.Files, minioRepo, log),
			}
		}

		response := shutdown.GroupedResponseWithURLs{
			Ges:   make([]*shutdown.ResponseWithURLs, 0),
			Mini:  make([]*shutdown.ResponseWithURLs, 0),
			Micro: make([]*shutdown.ResponseWithURLs, 0),
			Other: make([]*shutdown.ResponseWithURLs, 0),
		}

		for _, s := range shutdowns {
			types, ok := orgTypesMap[s.OrganizationID]
			if !ok {
				response.Other = append(response.Other, transformShutdown(s))
				continue
			}

			wasGrouped := false
			for _, t := range types {
				switch t {
				case "ges":
					response.Ges = append(response.Ges, transformShutdown(s))
					wasGrouped = true
				case "mini":
					response.Mini = append(response.Mini, transformShutdown(s))
					wasGrouped = true
				case "micro":
					response.Micro = append(response.Micro, transformShutdown(s))
					wasGrouped = true
				}
			}
			if !wasGrouped {
				response.Other = append(response.Other, transformShutdown(s))
			}
		}

		log.Info("successfully retrieved and grouped shutdowns", slog.Int("count", len(shutdowns)))
		render.JSON(w, r, response)
	}
}
