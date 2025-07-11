package edit

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
)

type Request struct {
	ID          int64  `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description,omitempty"`
}

type Response struct {
	resp.Response
}

type RoleEditor interface {
	EditRole(ctx context.Context, id int64, name, description string) (int64, error)
}

func New(log *slog.Logger, editor RoleEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.role.add.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		// Decode JSON
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		log.Info("request parsed", slog.Any("req", req))

		// Validate fields
		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)

			log.Error("failed to validate request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to validate request"))

			return
		}

		// edit role
		id, err := editor.EditRole(r.Context(), req.ID, req.Name, req.Description)
		if err != nil {
			log.Info("failed to edit role", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to add role"))
			return
		}

		log.Info("role successfully edited", slog.Int64("id", id))

		render.JSON(w, r, resp.Created())
	}
}
