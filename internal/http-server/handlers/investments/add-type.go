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

type addTypeRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description,omitempty"`
}

type addTypeResponse struct {
	resp.Response
	ID int `json:"id"`
}

type investmentTypeAdder interface {
	AddInvestmentType(ctx context.Context, req dto.AddInvestmentTypeRequest) (int, error)
}

func AddType(log *slog.Logger, adder investmentTypeAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.add-type"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req addTypeRequest
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
		storageReq := dto.AddInvestmentTypeRequest{
			Name:        req.Name,
			Description: req.Description,
		}

		// Create investment type
		id, err := adder.AddInvestmentType(r.Context(), storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("investment type name already exists", slog.String("name", req.Name))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Investment type with this name already exists"))
				return
			}
			log.Error("failed to add investment type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add investment type"))
			return
		}

		log.Info("investment type added successfully", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addTypeResponse{
			Response: resp.Created(),
			ID:       id,
		})
	}
}
