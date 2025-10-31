package patch

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"
)

type Request struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type PositionEditor interface {
	EditPosition(ctx context.Context, id int64, name, description *string) error
}

func New(log *slog.Logger, editor PositionEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.positions.patch.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Получаем ID из URL
		positionID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid position ID format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid position ID"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = editor.EditPosition(r.Context(), positionID, req.Name, req.Description)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("position not found", "id", positionID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Position not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("position name conflict", "name", *req.Name)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Position with this name already exists"))
				return
			}
			log.Error("failed to edit position", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to edit position"))
			return
		}

		log.Info("position updated successfully", slog.Int64("id", positionID))
		render.Status(r, http.StatusNoContent)
	}
}
