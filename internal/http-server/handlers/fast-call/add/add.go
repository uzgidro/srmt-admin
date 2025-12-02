package add

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

type addRequest struct {
	ContactID int64 `json:"contact_id" validate:"required"`
	Position  int   `json:"position" validate:"required"`
}

type addResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type fastCallAdder interface {
	AddFastCall(ctx context.Context, req dto.AddFastCallRequest) (int64, error)
}

func New(log *slog.Logger, adder fastCallAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.fast_call.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

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

		// Create fast call
		fastCallReq := dto.AddFastCallRequest{
			ContactID: req.ContactID,
			Position:  req.Position,
		}

		id, err := adder.AddFastCall(r.Context(), fastCallReq)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation", "contact_id", req.ContactID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Contact with this ID does not exist"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("duplicate position", "position", req.Position)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Fast call with this position already exists"))
				return
			}
			log.Error("failed to add fast call", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add fast call"))
			return
		}

		log.Info("fast call added successfully", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, addResponse{Response: resp.Created(), ID: id})
	}
}
