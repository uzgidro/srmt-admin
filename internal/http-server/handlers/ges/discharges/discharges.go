package discharges

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/discharge"
	"srmt-admin/internal/lib/model/organization"
	"srmt-admin/internal/lib/model/user"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

const layout = "2006-01-02" // YYYY-MM-DD

// DischargeGetter defines the interface for getting discharges by organization ID
type DischargeGetter interface {
	GetDischargesByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]discharge.Model, error)
}

// ResponseWithURLs is the API response model with presigned file URLs
type ResponseWithURLs struct {
	ID             int64               `json:"id"`
	Organization   *organization.Model `json:"organization"`
	CreatedByUser  *user.ShortInfo     `json:"created_by"`
	ApprovedByUser *user.ShortInfo     `json:"approved_by"`
	StartedAt      time.Time           `json:"started_at"`
	EndedAt        *time.Time          `json:"ended_at"`
	FlowRate       float64             `json:"flow_rate"`
	TotalVolume    float64             `json:"total_volume"`
	Reason         *string             `json:"reason"`
	IsOngoing      bool                `json:"is_ongoing"`
	Approved       *bool               `json:"approved"`
	Files          []dto.FileResponse  `json:"files,omitempty"`
}

// New creates a handler for GET /ges/{id}/discharges
func New(log *slog.Logger, getter DischargeGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.discharges.New"
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

		dischargesList, err := getter.GetDischargesByOrgID(r.Context(), orgID, startDate, endDate)
		if err != nil {
			log.Error("failed to get discharges", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve discharges"))
			return
		}

		// Transform to response with presigned URLs
		result := make([]ResponseWithURLs, 0, len(dischargesList))
		for _, d := range dischargesList {
			result = append(result, ResponseWithURLs{
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
			})
		}

		log.Info("successfully retrieved discharges", slog.Int("count", len(result)))
		render.JSON(w, r, result)
	}
}
