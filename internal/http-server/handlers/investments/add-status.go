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

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addStatusRequest struct {
	Name         string `json:"name" validate:"required,min=1,max=255"`
	Description  string `json:"description,omitempty"`
	TypeID       *int   `json:"type_id,omitempty"` // NULL = shared status
	DisplayOrder int    `json:"display_order" validate:"gte=0"`
}

type addStatusResponse struct {
	resp.Response
	ID int `json:"id"`
}

type investmentStatusAdder interface {
	AddInvestmentStatus(ctx context.Context, req dto.AddInvestmentStatusRequest) (int, error)
}

func AddStatus(log *slog.Logger, adder investmentStatusAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.add-status"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req addStatusRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Create DTO for storage
		storageReq := dto.AddInvestmentStatusRequest{
			Name:         req.Name,
			Description:  req.Description,
			TypeID:       req.TypeID,
			DisplayOrder: req.DisplayOrder,
		}

		// Create investment status
		id, err := adder.AddInvestmentStatus(r.Context(), storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("investment status name already exists for this type", slog.String("name", req.Name))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Investment status with this name already exists for this type"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("invalid type_id", slog.Any("type_id", req.TypeID))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid type_id"))
				return
			}
			log.Error("failed to add investment status", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add investment status"))
			return
		}

		log.Info("investment status added successfully", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addStatusResponse{
			Response: resp.Created(),
			ID:       id,
		})
	}
}
