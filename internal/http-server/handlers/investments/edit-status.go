package investments

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editStatusRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	TypeID       *int    `json:"type_id,omitempty"` // NULL = shared status
	DisplayOrder *int    `json:"display_order,omitempty"`
}

type investmentStatusEditor interface {
	EditInvestmentStatus(ctx context.Context, id int, req dto.EditInvestmentStatusRequest) error
}

func EditStatus(log *slog.Logger, editor investmentStatusEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.edit-status"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editStatusRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Build storage request
		storageReq := dto.EditInvestmentStatusRequest{
			Name:         req.Name,
			Description:  req.Description,
			TypeID:       req.TypeID,
			DisplayOrder: req.DisplayOrder,
		}

		// Update investment status
		err = editor.EditInvestmentStatus(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("investment status not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Investment status not found"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("investment status name already exists for this type")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Investment status with this name already exists for this type"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("invalid type_id")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid type_id"))
				return
			}
			log.Error("failed to update investment status", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update investment status"))
			return
		}

		log.Info("investment status updated successfully", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}
