package get_all

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/contact"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// ContactGetter - интерфейс для получения списка
type ContactGetter interface {
	GetAllContacts(ctx context.Context, filters dto.GetAllContactsFilters) ([]*contact.Model, error)
}

func New(log *slog.Logger, getter ContactGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.get_all.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Парсим фильтры
		var filters dto.GetAllContactsFilters
		q := r.URL.Query()

		if orgIDStr := q.Get("organization_id"); orgIDStr != "" {
			val, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'organization_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
				return
			}
			filters.OrganizationID = &val
		}

		if deptIDStr := q.Get("department_id"); deptIDStr != "" {
			val, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'department_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'department_id' parameter"))
				return
			}
			filters.DepartmentID = &val
		}

		// 2. Вызываем метод репозитория
		contacts, err := getter.GetAllContacts(r.Context(), filters)
		if err != nil {
			log.Error("failed to get all contacts", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve contacts"))
			return
		}

		// 3. Generate presigned URLs for icons
		for _, c := range contacts {
			if c.Icon != nil && c.Icon.URL != "" {
				presignedURL, err := minioRepo.GetPresignedURL(r.Context(), c.Icon.URL, 24*time.Hour)
				if err != nil {
					log.Error("failed to generate presigned URL for icon",
						slog.Int64("contact_id", c.ID),
						slog.String("object_key", c.Icon.URL),
						sl.Err(err))
					// Continue with empty URL instead of failing
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
