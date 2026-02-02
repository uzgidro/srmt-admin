package shutdowns

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/shutdown"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

const layout = "2006-01-02" // YYYY-MM-DD

// ShutdownGetter defines the interface for getting shutdowns by organization ID
type ShutdownGetter interface {
	GetShutdownsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*shutdown.ResponseModel, error)
}

// ResponseWithURLs is the API response model with presigned file URLs
type ResponseWithURLs struct {
	ID                            int64              `json:"id"`
	OrganizationID                int64              `json:"organization_id"`
	OrganizationName              string             `json:"organization_name"`
	StartedAt                     time.Time          `json:"started_at"`
	EndedAt                       *time.Time         `json:"ended_at,omitempty"`
	Reason                        *string            `json:"reason,omitempty"`
	CreatedByUser                 interface{}        `json:"created_by"`
	GenerationLossMwh             *float64           `json:"generation_loss,omitempty"`
	CreatedAt                     time.Time          `json:"created_at"`
	IdleDischargeVolumeThousandM3 *float64           `json:"idle_discharge_volume,omitempty"`
	Files                         []dto.FileResponse `json:"files,omitempty"`
}

// New creates a handler for GET /ges/{id}/shutdowns
func New(log *slog.Logger, getter ShutdownGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.shutdowns.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		orgID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		log = log.With(slog.Int64("organization_id", orgID))

		// Parse date filters
		var startDate, endDate *time.Time

		if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
			t, err := time.ParseInLocation(layout, startDateStr, loc)
			if err != nil {
				log.Warn("invalid 'start_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'start_date' format, use YYYY-MM-DD"))
				return
			}
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
			endDate = &t
		}

		shutdownsList, err := getter.GetShutdownsByOrgID(r.Context(), orgID, startDate, endDate)
		if err != nil {
			log.Error("failed to get shutdowns", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve shutdowns"))
			return
		}

		// Transform to response with presigned URLs
		result := make([]ResponseWithURLs, 0, len(shutdownsList))
		for _, s := range shutdownsList {
			result = append(result, ResponseWithURLs{
				ID:                            s.ID,
				OrganizationID:                s.OrganizationID,
				OrganizationName:              s.OrganizationName,
				StartedAt:                     s.StartedAt,
				EndedAt:                       s.EndedAt,
				Reason:                        s.Reason,
				CreatedByUser:                 s.CreatedByUser,
				GenerationLossMwh:             s.GenerationLossMwh,
				CreatedAt:                     s.CreatedAt,
				IdleDischargeVolumeThousandM3: s.IdleDischargeVolumeThousandM3,
				Files:                         helpers.TransformFilesWithURLs(r.Context(), s.Files, minioRepo, log),
			})
		}

		log.Info("successfully retrieved shutdowns", slog.Int("count", len(result)))
		render.JSON(w, r, result)
	}
}
