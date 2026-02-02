package contacts

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/contact"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// ContactGetter defines the interface for getting contacts by organization ID
type ContactGetter interface {
	GetContactsByOrgID(ctx context.Context, orgID int64) ([]*contact.Model, error)
}

// New creates a handler for GET /ges/{id}/contacts
func New(log *slog.Logger, getter ContactGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges.contacts.New"
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

		contacts, err := getter.GetContactsByOrgID(r.Context(), orgID)
		if err != nil {
			log.Error("failed to get contacts", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve contacts"))
			return
		}

		// Generate presigned URLs for icons
		for _, c := range contacts {
			if c.Icon != nil && c.Icon.URL != "" {
				presignedURL, err := minioRepo.GetPresignedURL(r.Context(), c.Icon.URL, 24*time.Hour)
				if err != nil {
					log.Error("failed to generate presigned URL for icon",
						slog.Int64("contact_id", c.ID),
						slog.String("object_key", c.Icon.URL),
						sl.Err(err))
					c.Icon.URL = ""
				} else {
					c.Icon.URL = presignedURL.String()
				}
			}
		}

		log.Info("successfully retrieved contacts", slog.Int("count", len(contacts)))
		render.JSON(w, r, contacts)
	}
}
