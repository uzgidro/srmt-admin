package visits

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/lib/model/visit"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

const layout = "2006-01-02" // YYYY-MM-DD

// VisitGetter defines the interface for getting visits by organization ID
type VisitGetter interface {
	GetVisitsByOrgID(ctx context.Context, orgID int64, startDate, endDate *time.Time) ([]*visit.ResponseModel, error)
}

// ResponseWithURLs is the API response model with presigned file URLs
type ResponseWithURLs struct {
	ID               int64              `json:"id"`
	OrganizationID   int64              `json:"organization_id"`
	OrganizationName string             `json:"organization_name"`
	VisitDate        time.Time          `json:"visit_date"`
	Description      string             `json:"description"`
	ResponsibleName  string             `json:"responsible_name"`
	CreatedAt        time.Time          `json:"created_at"`
	CreatedByUser    *user.ShortInfo    `json:"created_by"`
	Files            []dto.FileResponse `json:"files,omitempty"`
}

// New creates a handler for GET /ges/{id}/visits
func New(log *slog.Logger, getter VisitGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.visits.New"
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

		visitsList, err := getter.GetVisitsByOrgID(r.Context(), orgID, startDate, endDate)
		if err != nil {
			log.Error("failed to get visits", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve visits"))
			return
		}

		// Transform to response with presigned URLs
		result := make([]ResponseWithURLs, 0, len(visitsList))
		for _, v := range visitsList {
			result = append(result, ResponseWithURLs{
				ID:               v.ID,
				OrganizationID:   v.OrganizationID,
				OrganizationName: v.OrganizationName,
				VisitDate:        v.VisitDate,
				Description:      v.Description,
				ResponsibleName:  v.ResponsibleName,
				CreatedAt:        v.CreatedAt,
				CreatedByUser:    v.CreatedByUser,
				Files:            helpers.TransformFilesWithURLs(r.Context(), v.Files, minioRepo, log),
			})
		}

		log.Info("successfully retrieved visits", slog.Int("count", len(result)))
		render.JSON(w, r, result)
	}
}
