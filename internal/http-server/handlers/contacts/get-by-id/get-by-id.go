package get_by_id

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/storage"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// ContactGetter - интерфейс для получения одного
type ContactGetter interface {
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)
}

func New(log *slog.Logger, getter ContactGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.get_by_id.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		contactModel, err := getter.GetContactByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("contact not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Contact not found"))
				return
			}
			log.Error("failed to get contact", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve contact"))
			return
		}

		// Generate presigned URL for icon if present
		if contactModel.Icon != nil && contactModel.Icon.URL != "" {
			presignedURL, err := minioRepo.GetPresignedURL(r.Context(), contactModel.Icon.URL, 24*time.Hour)
			if err != nil {
				log.Error("failed to generate presigned URL for icon",
					slog.Int64("contact_id", contactModel.ID),
					slog.String("object_key", contactModel.Icon.URL),
					sl.Err(err))
				// Continue with empty URL instead of failing the entire request
				contactModel.Icon.URL = ""
			} else {
				contactModel.Icon.URL = presignedURL.String()
			}
		}

		log.Info("successfully retrieved contact", slog.Int64("id", contactModel.ID))
		render.JSON(w, r, contactModel)
	}
}
