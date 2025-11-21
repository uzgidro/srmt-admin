package delete

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/storage"
)

type MetaDataDeleter interface {
	GetFileByID(ctx context.Context, id int64) (file.Model, error)
	DeleteFile(ctx context.Context, id int64) error
}

type FileDeleter interface {
	DeleteFile(ctx context.Context, objectName string) error
}

func New(log *slog.Logger, dataDeleter MetaDataDeleter, fileDeleter FileDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.delete.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		fileIDStr := chi.URLParam(r, "fileID")
		fileID, err := strconv.ParseInt(fileIDStr, 10, 64)
		if err != nil {
			log.Warn("invalid file ID format", sl.Err(err), slog.String("file_id", fileIDStr))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid file ID"))
			return
		}

		fileMeta, err := dataDeleter.GetFileByID(r.Context(), fileID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("file not found in db, nothing to delete", slog.Int64("file_id", fileID))
				render.Status(r, http.StatusNoContent)
				return
			}
			log.Error("failed to get file metadata before deletion", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve file data for deletion"))
			return
		}

		if err := dataDeleter.DeleteFile(r.Context(), fileID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("file already deleted from db", slog.Int64("file_id", fileID))
				render.Status(r, http.StatusNoContent)
				return
			}
			log.Error("failed to delete file record from db", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete file from database"))
			return
		}
		log.Info("file record deleted from database", slog.Int64("file_id", fileID))

		if err := fileDeleter.DeleteFile(r.Context(), fileMeta.ObjectKey); err != nil {
			log.Error("CRITICAL: failed to delete file from minio storage after deleting from db",
				sl.Err(err),
				slog.String("object_key", fileMeta.ObjectKey),
			)
		} else {
			log.Info("file deleted from minio storage", slog.String("object_key", fileMeta.ObjectKey))
		}

		render.Status(r, http.StatusNoContent)
	}
}
