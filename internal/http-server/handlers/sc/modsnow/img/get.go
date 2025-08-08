package img

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage/minio"
)

// Lister defines the interface for listing image URLs from a storage provider.
type Lister interface {
	ListImageURLs(ctx context.Context, bucketName string) ([]minio.ImageURL, error)
}

// Get creates a generic HTTP handler for listing images from a specific bucket.
func Get(log *slog.Logger, lister Lister, bucketName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sc.modsnow.img.Get"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("bucket", bucketName),
		)

		urls, err := lister.ListImageURLs(r.Context(), bucketName)
		if err != nil {
			log.Error("failed to list image urls", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("could not retrieve images"))
			return
		}

		log.Info("successfully retrieved image urls", slog.Int("count", len(urls)))

		render.JSON(w, r, urls)
	}
}
