package add

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// Request (JSON DTO)
type addRequest struct {
	Name        string    `json:"name" validate:"required"`
	Together    *string   `json:"together,omitempty"`
	Date        time.Time `json:"date" validate:"required"`
	Description *string   `json:"description,omitempty"`
	Visitor     string    `json:"visitor" validate:"required"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type receptionAdder interface {
	AddReception(ctx context.Context, req dto.AddReceptionRequest) (int64, error)
}

func New(log *slog.Logger, adder receptionAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reception.add.New"
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

		// Create reception
		receptionReq := dto.AddReceptionRequest{
			Name:        req.Name,
			Together:    req.Together,
			Date:        req.Date,
			Description: req.Description,
			Visitor:     req.Visitor,
			CreatedByID: userID,
		}

		id, err := adder.AddReception(r.Context(), receptionReq)
		if err != nil {
			log.Error("failed to add reception", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add reception"))
			return
		}

		log.Info("reception added successfully", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{Response: resp.Created(), ID: id})
	}
}
