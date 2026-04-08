package investments

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addRequest struct {
	Name     string  `json:"name" validate:"required"`
	TypeID   int     `json:"type_id" validate:"required,min=1"`
	StatusID int     `json:"status_id" validate:"required,min=1"`
	Cost     float64 `json:"cost" validate:"gte=0"`
	Comments *string `json:"comments,omitempty"`
	FileIDs  []int64 `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type investmentAdder interface {
	AddInvestment(ctx context.Context, req dto.AddInvestmentRequest, createdByID int64) (int64, error)
	LinkInvestmentFiles(ctx context.Context, investmentID int64, fileIDs []int64) error
}

func Add(log *slog.Logger, adder investmentAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
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
		storageReq := dto.AddInvestmentRequest{
			Name:     req.Name,
			TypeID:   req.TypeID,
			StatusID: req.StatusID,
			Cost:     req.Cost,
			Comments: req.Comments,
			FileIDs:  req.FileIDs,
		}

		// Create investment
		id, err := adder.AddInvestment(r.Context(), storageReq, userID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation", slog.Int("status_id", req.StatusID))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid status ID"))
				return
			}
			log.Error("failed to add investment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add investment"))
			return
		}

		// Link files if provided
		if len(req.FileIDs) > 0 {
			if err := adder.LinkInvestmentFiles(r.Context(), id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
				// Don't fail the request, just log the error
			}
		}

		log.Info("investment added successfully",
			slog.Int64("id", id),
			slog.Int("total_files", len(req.FileIDs)),
		)

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{
			Response: resp.Created(),
			ID:       id,
		})
	}
}
