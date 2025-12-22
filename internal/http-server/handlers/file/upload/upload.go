package upload

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/lib/model/file"
	"strconv"
	"time"
)

type FileUploader interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	DeleteFile(ctx context.Context, objectName string) error
}

// FileMetaSaver определяет интерфейс для сохранения метаданных файла в БД.
type FileMetaSaver interface {
	AddFile(ctx context.Context, fileData file.Model) (int64, error)
	GetCategoryByID(ctx context.Context, id int64) (category.Model, error)
}

// New создает новый HTTP-хендлер для загрузки файлов.
// bucketName - это название бакета в MinIO, куда будут загружаться файлы.
func New(log *slog.Logger, uploader FileUploader, saver FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.upload.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Устанавливаем лимит на размер тела запроса (например, 50 MB) и парсим форму.
		const maxUploadSize = 50 * 1024 * 1024
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			log.Error("failed to parse multipart form or file is too large", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request or file is too large"))
			return
		}

		// 2. Получаем файл из формы по ключу "file".
		formFile, handler, err := r.FormFile("file")
		if err != nil {
			log.Error("failed to get file from form", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Form field 'file' is required"))
			return
		}
		defer formFile.Close()

		// 3. Получаем ID категории из формы.
		categoryIDStr := r.FormValue("category_id")
		categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil {
			log.Error("invalid category_id", sl.Err(err), slog.String("value", categoryIDStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid or missing form field 'category_id'"))
			return
		}

		cat, err := saver.GetCategoryByID(r.Context(), categoryID)
		if err != nil {
			log.Warn("failed to get category", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Incorrect category"))
		}

		var fileDate time.Time
		dateStr := r.FormValue("date") // Ожидаем формат "YYYY-MM-DD"

		if dateStr == "" {
			// Если дата не предоставлена, используем текущую.
			fileDate = time.Now()
			log.Info("date not provided, using current date")
		} else {
			// Если дата предоставлена, парсим ее.
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Error("invalid date format", sl.Err(err), slog.String("date_value", dateStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid date format. Please use YYYY-MM-DD."))
				return
			}
			fileDate = parsedDate
			log.Info("using provided date", slog.String("date", dateStr))
		}

		// 4. Генерируем уникальное имя для объекта в MinIO, чтобы избежать конфликтов.
		// Формат: <category>/<date>/<uuid>.<ext>
		datePrefix := fileDate.Format("2006/01/02")
		objectKey := fmt.Sprintf("%s/%s/%s%s",
			cat.DisplayName,
			datePrefix,
			uuid.New().String(),
			filepath.Ext(handler.Filename),
		)

		// 5. Загружаем файл в MinIO.
		// Это первая часть нашей "транзакции".
		err = uploader.UploadFile(r.Context(), objectKey, formFile, handler.Size, handler.Header.Get("Content-Type"))
		if err != nil {
			log.Error("failed to upload file to storage", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Could not upload file to storage"))
			return
		}

		// 6. Если загрузка в MinIO прошла успешно, сохраняем метаданные в PostgreSQL.
		fileModel := file.Model{
			FileName:   handler.Filename,
			ObjectKey:  objectKey,
			CategoryID: categoryID,
			MimeType:   handler.Header.Get("Content-Type"),
			SizeBytes:  handler.Size,
			CreatedAt:  time.Now(),
			TargetDate: fileDate,
		}

		fileID, err := saver.AddFile(r.Context(), fileModel)
		if err != nil {
			log.Error("failed to save file metadata to database", sl.Err(err))
			if delErr := uploader.DeleteFile(r.Context(), objectKey); delErr != nil {
				log.Error("COMPENSATION FAILED: could not delete orphaned file from storage",
					sl.Err(delErr),
					slog.String("object_key", objectKey),
				)
			} else {
				log.Info("compensation successful: orphaned file deleted from storage", slog.String("object_key", objectKey))
			}

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Could not save file metadata"))
			return
		}

		log.Info("file uploaded successfully", slog.Int64("id", fileID), slog.String("object_key", objectKey))

		render.JSON(w, r, resp.Created())
	}
}
