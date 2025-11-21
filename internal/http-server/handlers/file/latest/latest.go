package latest

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/file"
)

type FileGetter interface {
	GetLatestFiles(ctx context.Context) ([]file.LatestFile, error)
}

type URLGetter interface {
	GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error)
}

type ResponseItem struct {
	ID           int64     `json:"id"`
	FileName     string    `json:"file_name"`
	Extension    string    `json:"extension"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
	CategoryName string    `json:"category_name"`
	URL          string    `json:"url"`
}

func New(log *slog.Logger, fileGetter FileGetter, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.latest.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		latestFiles, err := fileGetter.GetLatestFiles(r.Context())
		if err != nil {
			log.Error("failed to get latest files", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve latest files"))
			return
		}

		if len(latestFiles) == 0 {
			log.Info("no latest files found")
			render.JSON(w, r, []ResponseItem{})
			return
		}

		responseItems := make([]ResponseItem, 0, len(latestFiles))
		for _, f := range latestFiles {
			u, err := urlGetter.GetPresignedURL(r.Context(), f.ObjectKey, time.Hour)
			if err != nil {
				log.Error("failed to get presigned URL", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to retrieve presigned URL"))
				return
			}

			responseItems = append(responseItems, ResponseItem{
				ID:           f.ID,
				FileName:     f.FileName,
				Extension:    f.GetExtension(),
				SizeBytes:    f.SizeBytes,
				CreatedAt:    f.CreatedAt,
				CategoryName: f.CategoryName,
				URL:          u.String(),
			})
		}

		log.Info("successfully retrieved latest files", slog.Int("count", len(responseItems)))
		render.JSON(w, r, responseItems)
	}
}
