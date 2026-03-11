package locations

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/filtration"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type LocationCreator interface {
	CreateFiltrationLocation(ctx context.Context, req filtration.CreateLocationRequest, userID int64) (int64, error)
}

func Add(log *slog.Logger, creator LocationCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.locations.Add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req filtration.CreateLocationRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)
			log.Error("failed to validate request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(validationErrors))
			return
		}

		id, err := creator.CreateFiltrationLocation(r.Context(), req, userID)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("location already exists", sl.Err(err))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Location already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("invalid organization_id", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
				return
			}
			log.Error("failed to create location", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create location"))
			return
		}

		log.Info("location created", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp.Created())
	}
}
