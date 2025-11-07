package add

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
)

// Request определяет структуру для входящего JSON-запроса.
type Request struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
}

// Response определяет структуру для успешного ответа.
type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

// PositionAdder определяет интерфейс для добавления должности.
type PositionAdder interface {
	AddPosition(ctx context.Context, name string, description *string) (int64, error)
}

func New(log *slog.Logger, adder PositionAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.position.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := adder.AddPosition(r.Context(), req.Name, req.Description)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("position name duplicate", "name", req.Name)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Position with this name already exists"))
				return
			}
			log.Error("failed to add position", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add position"))
			return
		}

		log.Info("position added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: id})
	}
}
