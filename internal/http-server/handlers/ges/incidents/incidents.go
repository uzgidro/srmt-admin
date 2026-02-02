package incidents

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/incident"
	"srmt-admin/internal/lib/model/user"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

const layout = "2006-01-02" // YYYY-MM-DD

// IncidentGetter defines the interface for getting incidents by organization ID
type IncidentGetter interface {
	GetIncidentsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*incident.ResponseModel, error)
}

// ResponseWithURLs is the API response model with presigned file URLs
type ResponseWithURLs struct {
	ID               int64              `json:"id"`
	IncidentTime     time.Time          `json:"incident_date"`
	Description      string             `json:"description"`
	CreatedAt        time.Time          `json:"created_at"`
	OrganizationID   *int64             `json:"organization_id,omitempty"`
	OrganizationName *string            `json:"organization,omitempty"`
	CreatedByUser    *user.ShortInfo    `json:"created_by"`
	Files            []dto.FileResponse `json:"files,omitempty"`
}

// New creates a handler for GET /ges/{id}/incidents
func New(log *slog.Logger, getter IncidentGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.incidents.New"
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

		incidentsList, err := getter.GetIncidentsByOrgID(r.Context(), orgID, startDate, endDate)
		if err != nil {
			log.Error("failed to get incidents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve incidents"))
			return
		}

		// Transform to response with presigned URLs
		result := make([]ResponseWithURLs, 0, len(incidentsList))
		for _, inc := range incidentsList {
			result = append(result, ResponseWithURLs{
				ID:               inc.ID,
				IncidentTime:     inc.IncidentTime,
				Description:      inc.Description,
				CreatedAt:        inc.CreatedAt,
				OrganizationID:   inc.OrganizationID,
				OrganizationName: inc.OrganizationName,
				CreatedByUser:    inc.CreatedByUser,
				Files:            helpers.TransformFilesWithURLs(r.Context(), inc.Files, minioRepo, log),
			})
		}

		log.Info("successfully retrieved incidents", slog.Int("count", len(result)))
		render.JSON(w, r, result)
	}
}
