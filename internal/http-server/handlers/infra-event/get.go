package infraevent

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	infraeventmodel "srmt-admin/internal/lib/model/infra-event"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type eventGetter interface {
	GetInfraEvents(ctx context.Context, categoryID int64, day time.Time) ([]*infraeventmodel.ResponseModel, error)
	GetInfraEventsByDate(ctx context.Context, day time.Time) ([]*infraeventmodel.ResponseModel, error)
}

const dateLayout = "2006-01-02"

func Get(log *slog.Logger, getter eventGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event.get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse date
		var day time.Time
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			now := time.Now().In(loc)
			day = time.Date(now.Year(), now.Month(), now.Day(), 5, 0, 0, 0, loc)
		} else {
			t, err := time.ParseInLocation(dateLayout, dateStr, loc)
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'date' format, use YYYY-MM-DD"))
				return
			}
			day = time.Date(t.Year(), t.Month(), t.Day(), 5, 0, 0, 0, loc)
		}

		// Parse optional category_id
		var events []*infraeventmodel.ResponseModel
		var err error

		categoryIDStr := r.URL.Query().Get("category_id")
		if categoryIDStr != "" {
			categoryID, parseErr := strconv.ParseInt(categoryIDStr, 10, 64)
			if parseErr != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'category_id' parameter"))
				return
			}
			events, err = getter.GetInfraEvents(r.Context(), categoryID, day)
		} else {
			events, err = getter.GetInfraEventsByDate(r.Context(), day)
		}

		if err != nil {
			log.Error("failed to get infra events", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve events"))
			return
		}

		// Transform to include presigned URLs
		result := make([]*infraeventmodel.ResponseWithURLs, 0, len(events))
		for _, e := range events {
			result = append(result, &infraeventmodel.ResponseWithURLs{
				ID:               e.ID,
				CategoryID:       e.CategoryID,
				CategorySlug:     e.CategorySlug,
				CategoryName:     e.CategoryName,
				OrganizationID:   e.OrganizationID,
				OrganizationName: e.OrganizationName,
				OccurredAt:       e.OccurredAt,
				RestoredAt:       e.RestoredAt,
				Description:      e.Description,
				Remediation:      e.Remediation,
				Notes:            e.Notes,
				CreatedAt:        e.CreatedAt,
				CreatedByUser:    e.CreatedByUser,
				Files:            helpers.TransformFilesWithURLs(r.Context(), e.Files, minioRepo, log),
			})
		}

		render.JSON(w, r, result)
	}
}
