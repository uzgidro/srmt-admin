package letters

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/letter"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type letterByIDGetter interface {
	GetLetterByID(ctx context.Context, id int64) (*letter.ResponseModel, error)
}

func GetByID(log *slog.Logger, getter letterByIDGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.letters.get-by-id"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := formparser.GetURLParamInt64(r, "id")
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		doc, err := getter.GetLetterByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("letter not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Letter not found"))
				return
			}
			log.Error("failed to get letter", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve letter"))
			return
		}

		docWithURLs := transformLetterToResponse(r.Context(), doc, minioRepo, log)

		log.Info("successfully retrieved letter", slog.Int64("id", id))
		render.JSON(w, r, docWithURLs)
	}
}
