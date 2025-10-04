package download

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/storage"
)

type FileMetaGetter interface {
	GetFileByID(ctx context.Context, id int64) (file.Model, error)
}

type PresignedURLGenerator interface {
	GetPresignedURL(ctx context.Context, objectName string, expires time.Duration) (*url.URL, error)
}

func New(log *slog.Logger, metaGetter FileMetaGetter, urlGenerator PresignedURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.download.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Получаем ID файла из URL.
		fileIDStr := chi.URLParam(r, "fileID")
		fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
		if err != nil {
			log.Warn("invalid file ID format", sl.Err(err), slog.String("file_id", fileIDStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid file ID"))
			return
		}

		// 2. Получаем метаданные файла из БД.
		fileMeta, err := metaGetter.GetFileByID(r.Context(), fileID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("file not found in db", slog.Int64("file_id", fileID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("File not found"))
				return
			}
			log.Error("failed to get file metadata", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve file data"))
			return
		}

		// 3. Генерируем временную ссылку для скачивания (например, на 5 минут).
		expires := 5 * time.Minute
		presignedURL, err := urlGenerator.GetPresignedURL(r.Context(), fileMeta.ObjectKey, expires)
		if err != nil {
			log.Error("failed to generate presigned URL", sl.Err(err), slog.String("object_key", fileMeta.ObjectKey))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Could not generate download link"))
			return
		}

		log.Info("successfully generated presigned URL", slog.Int64("file_id", fileID), slog.String("object_key", fileMeta.ObjectKey))

		// 4. Отправляем клиенту редирект на сгенерированную ссылку.
		// Браузер автоматически последует по этому адресу и начнет скачивание.
		http.Redirect(w, r, presignedURL.String(), http.StatusTemporaryRedirect)
	}
}
