package stock

import (
	"bytes"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
)

const (
	formFileKey = "file"
	maxMemory   = 32 << 20
	targetURL   = "http://localhost:19789/parse-stock"
)

// Upload теперь просто пересылает файл и ожидает подтверждения о приеме.
func Upload(log *slog.Logger, client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sc.stock.Upload"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Парсим и извлекаем файл (без изменений)
		if err := r.ParseMultipartForm(maxMemory); err != nil {
			log.Error("failed to parse multipart form", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse form"))
			return
		}
		file, header, err := r.FormFile(formFileKey)
		if err != nil {
			log.Error("failed to get file from form", sl.Err(err), slog.String("key", formFileKey))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("file not found in request"))
			return
		}
		defer file.Close()
		log.Info("received file", slog.String("filename", header.Filename), slog.Int64("size", header.Size))

		// 2. Создаем тело нового запроса (без изменений)
		var requestBody bytes.Buffer
		multipartWriter := multipart.NewWriter(&requestBody)
		part, err := multipartWriter.CreateFormFile(formFileKey, header.Filename)
		if err != nil {
			log.Error("failed to create form file for proxy request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			log.Error("failed to copy file content to proxy request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}
		if err := multipartWriter.Close(); err != nil {
			log.Error("failed to close multipart writer", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}

		// 3. Отправляем запрос на целевой URL (без изменений)
		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, &requestBody)
		if err != nil {
			log.Error("failed to create proxy request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}
		proxyReq.Header.Set("Content-Type", multipartWriter.FormDataContentType())

		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			log.Error("failed to execute proxy request", sl.Err(err), slog.String("target_url", targetURL))
			render.Status(r, http.StatusBadGateway)
			render.JSON(w, r, resp.BadGateway("failed to reach target service"))
			return
		}
		defer proxyResp.Body.Close()

		// 4. Проверяем статус ответа от стороннего сервиса.
		// Мы ожидаем 200 OK или 202 Accepted как знак того, что файл успешно принят.
		if proxyResp.StatusCode >= 300 {
			log.Error("target service rejected the file",
				slog.Int("status_code", proxyResp.StatusCode),
				slog.String("target_url", targetURL),
			)
			// Проксируем ошибку клиенту
			w.WriteHeader(proxyResp.StatusCode)
			io.Copy(w, proxyResp.Body)
			return
		}

		log.Info("file successfully accepted by target service", slog.Int("status_code", proxyResp.StatusCode))

		// 5. Отвечаем клиенту, что его файл принят в обработку.
		render.Status(r, http.StatusAccepted)
		render.JSON(w, r, map[string]string{"message": "file accepted for processing"})
	}
}
