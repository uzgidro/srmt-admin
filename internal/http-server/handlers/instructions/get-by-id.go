package instructions

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/instruction"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type instructionByIDGetter interface {
	GetInstructionByID(ctx context.Context, id int64) (*instruction.ResponseModel, error)
}

func GetByID(log *slog.Logger, getter instructionByIDGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.instructions.get-by-id"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := formparser.GetURLParamInt64(r, "id")
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		doc, err := getter.GetInstructionByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("instruction not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Instruction not found"))
				return
			}
			log.Error("failed to get instruction", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve instruction"))
			return
		}

		docWithURLs := transformInstructionToResponse(r.Context(), doc, minioRepo, log)

		log.Info("successfully retrieved instruction", slog.Int64("id", id))
		render.JSON(w, r, docWithURLs)
	}
}
