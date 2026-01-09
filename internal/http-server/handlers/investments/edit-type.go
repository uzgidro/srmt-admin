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

type editTypeRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type investmentTypeEditor interface {
	EditInvestmentType(ctx context.Context, id int, req dto.EditInvestmentTypeRequest) error
}

func EditType(log *slog.Logger, editor investmentTypeEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.edit-type"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editTypeRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Build storage request
		storageReq := dto.EditInvestmentTypeRequest{
			Name:        req.Name,
			Description: req.Description,
		}

		// Update investment type
		err = editor.EditInvestmentType(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("investment type not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Investment type not found"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("investment type name already exists")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Investment type with this name already exists"))
				return
			}
			log.Error("failed to update investment type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update investment type"))
			return
		}

		log.Info("investment type updated successfully", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}
