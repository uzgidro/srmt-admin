package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/lib/model/file"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
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

type uploadedFile struct {
	ID       int64  `json:"id"`
	FileName string `json:"file_name"`
}

// New создает новый HTTP-хендлер для загрузки файлов.
// Поддерживает один файл (поле "file") или несколько (поле "files").
func New(log *slog.Logger, uploader FileUploader, saver FileMetaSaver, parserURL, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.upload.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Устанавливаем лимит на размер тела запроса (50 MB) и парсим форму.
		const maxUploadSize = 50 * 1024 * 1024
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			log.Error("failed to parse multipart form or file is too large", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request or file is too large"))
			return
		}

		// 2. Получаем ID категории из формы.
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
			return
		}

		// 3. Парсим дату.
		var fileDate time.Time
		dateStr := r.FormValue("date")
		if dateStr == "" {
			fileDate = time.Now()
		} else {
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Error("invalid date format", sl.Err(err), slog.String("date_value", dateStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid date format. Please use YYYY-MM-DD."))
				return
			}
			fileDate = parsedDate
		}

		// 4. Собираем файлы — поддерживаем "file" (один) и "files" (несколько).
		var fileHeaders []*multipart.FileHeader
		if r.MultipartForm != nil {
			if fh, ok := r.MultipartForm.File["files"]; ok {
				fileHeaders = append(fileHeaders, fh...)
			}
			if fh, ok := r.MultipartForm.File["file"]; ok {
				fileHeaders = append(fileHeaders, fh...)
			}
		}

		if len(fileHeaders) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Form field 'file' or 'files' is required"))
			return
		}

		// 5. Загружаем каждый файл.
		datePrefix := fileDate.Format("2006/01/02")
		var uploaded []uploadedFile
		var uploadedObjectKeys []string

		for _, fh := range fileHeaders {
			f, err := fh.Open()
			if err != nil {
				log.Error("failed to open file", sl.Err(err), slog.String("filename", fh.Filename))
				compensateUploads(r.Context(), log, uploader, saver, uploaded, uploadedObjectKeys)
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to open uploaded file"))
				return
			}

			objectKey := fmt.Sprintf("%s/%s/%s%s",
				cat.DisplayName,
				datePrefix,
				uuid.New().String(),
				filepath.Ext(fh.Filename),
			)

			err = uploader.UploadFile(r.Context(), objectKey, f, fh.Size, fh.Header.Get("Content-Type"))
			f.Close()
			if err != nil {
				log.Error("failed to upload file to storage", sl.Err(err), slog.String("filename", fh.Filename))
				compensateUploads(r.Context(), log, uploader, saver, uploaded, uploadedObjectKeys)
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Could not upload file to storage"))
				return
			}
			uploadedObjectKeys = append(uploadedObjectKeys, objectKey)

			fileModel := file.Model{
				FileName:   fh.Filename,
				ObjectKey:  objectKey,
				CategoryID: categoryID,
				MimeType:   fh.Header.Get("Content-Type"),
				SizeBytes:  fh.Size,
				CreatedAt:  time.Now(),
				TargetDate: fileDate,
			}

			fileID, err := saver.AddFile(r.Context(), fileModel)
			if err != nil {
				log.Error("failed to save file metadata", sl.Err(err), slog.String("filename", fh.Filename))
				compensateUploads(r.Context(), log, uploader, saver, uploaded, uploadedObjectKeys)
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Could not save file metadata"))
				return
			}

			uploaded = append(uploaded, uploadedFile{ID: fileID, FileName: fh.Filename})
			log.Info("file uploaded", slog.Int64("id", fileID), slog.String("object_key", objectKey))
		}

		// 6. Production parser — только для одного файла через "file".
		if cat.Name == "production" && len(fileHeaders) == 1 {
			sendToParser(r, log, fileHeaders[0], parserURL, apiKey)
		}

		// 7. Ответ.
		log.Info("upload complete", slog.Int("count", len(uploaded)))

		// Обратная совместимость: если один файл — вернуть id на верхнем уровне
		if len(uploaded) == 1 {
			render.Status(r, http.StatusCreated)
			render.JSON(w, r, struct {
				resp.Response
				ID       int64          `json:"id"`
				Uploaded []uploadedFile `json:"uploaded_files,omitempty"`
			}{
				Response: resp.Created(),
				ID:       uploaded[0].ID,
				Uploaded: uploaded,
			})
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, struct {
			resp.Response
			IDs      []int64        `json:"ids"`
			Uploaded []uploadedFile `json:"uploaded_files"`
		}{
			Response: resp.Created(),
			IDs:      extractIDs(uploaded),
			Uploaded: uploaded,
		})
	}
}

func extractIDs(files []uploadedFile) []int64 {
	ids := make([]int64, len(files))
	for i, f := range files {
		ids[i] = f.ID
	}
	return ids
}

// compensateUploads удаляет уже загруженные файлы при ошибке.
func compensateUploads(ctx context.Context, log *slog.Logger, uploader FileUploader, saver FileMetaSaver, uploaded []uploadedFile, objectKeys []string) {
	for _, key := range objectKeys {
		if err := uploader.DeleteFile(ctx, key); err != nil {
			log.Error("compensation: failed to delete file from storage", sl.Err(err), slog.String("object_key", key))
		}
	}
	// Метаданные в БД удалятся каскадно или останутся осиротевшими — логируем
	if len(uploaded) > 0 {
		log.Warn("compensation: orphaned file metadata may remain in DB", slog.Int("count", len(uploaded)))
	}
}

// sendToParser отправляет файл в prime-parser (для категории production).
func sendToParser(r *http.Request, log *slog.Logger, fh *multipart.FileHeader, parserURL, apiKey string) {
	f, err := fh.Open()
	if err != nil {
		log.Error("failed to open file for parser", sl.Err(err))
		return
	}
	defer f.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fh.Filename)
	if err != nil {
		log.Error("failed to create multipart writer for parser", sl.Err(err))
		return
	}
	if _, err := io.Copy(part, f); err != nil {
		log.Error("failed to copy file for parser", sl.Err(err))
		return
	}
	writer.Close()

	req, err := http.NewRequestWithContext(r.Context(), "POST", parserURL, body)
	if err != nil {
		log.Error("failed to create parser request", sl.Err(err))
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	respParser, err := client.Do(req)
	if err != nil {
		log.Error("failed to send file to parser", sl.Err(err))
		return
	}
	defer respParser.Body.Close()

	if respParser.StatusCode != http.StatusAccepted {
		log.Warn("parser returned unexpected status", slog.Int("status", respParser.StatusCode))
	} else {
		log.Info("file sent to parser successfully")
	}
}
