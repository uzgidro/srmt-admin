package getbycategory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type FileGetter interface {
	GetLatestFileByCategoryAndDate(ctx context.Context, categoryName string, targetDate string) (file.Model, error)
}

type PresignedURLGenerator interface {
	GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error)
}

type Response struct {
	ID         int64     `json:"id"`
	FileName   string    `json:"file_name"`
	SizeBytes  int64     `json:"size_bytes"`
	CategoryID int64     `json:"category_id"`
	MimeType   string    `json:"mime_type"`
	CreatedAt  time.Time `json:"created_at"`
	TargetDate time.Time `json:"target_date"`
	URL        string    `json:"url"`
}

func New(log *slog.Logger, fileGetter FileGetter, urlGenerator PresignedURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.get-by-category.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Get category name from query parameter
		categoryName := r.URL.Query().Get("category")
		if categoryName == "" {
			log.Warn("category parameter is required")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Query parameter 'category' is required"))
			return
		}

		// 2. Get date from query parameter, if not provided use today
		dateStr := r.URL.Query().Get("date")
		if dateStr == "" {
			// Use today's date in YYYY-MM-DD format
			dateStr = time.Now().Format("2006-01-02")
			log.Info("date not provided, using today", slog.String("date", dateStr))
		} else {
			// Validate date format
			_, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Warn("invalid date format", sl.Err(err), slog.String("date", dateStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid date format. Please use YYYY-MM-DD"))
				return
			}
		}

		// 3. Get latest file by category and date
		fileMeta, err := fileGetter.GetLatestFileByCategoryAndDate(r.Context(), categoryName, dateStr)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Info("no file found for category and date",
					slog.String("category", categoryName),
					slog.String("date", dateStr))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("No file found for the specified category and date"))
				return
			}
			log.Error("failed to get file", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve file"))
			return
		}

		// 4. Generate presigned URL for the file
		expires := 5 * time.Hour
		presignedURL, err := urlGenerator.GetPresignedURL(r.Context(), fileMeta.ObjectKey, expires)
		if err != nil {
			log.Error("failed to generate presigned URL", sl.Err(err), slog.String("object_key", fileMeta.ObjectKey))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Could not generate download link"))
			return
		}

		log.Info("successfully retrieved file by category and date",
			slog.Int64("file_id", fileMeta.ID),
			slog.String("category", categoryName),
			slog.String("date", dateStr))

		// 5. Return file metadata with presigned URL
		response := Response{
			ID:         fileMeta.ID,
			FileName:   fileMeta.FileName,
			SizeBytes:  fileMeta.SizeBytes,
			CategoryID: fileMeta.CategoryID,
			MimeType:   fileMeta.MimeType,
			CreatedAt:  fileMeta.CreatedAt,
			TargetDate: fileMeta.TargetDate,
			URL:        presignedURL.String(),
		}

		render.JSON(w, r, response)
	}
}
