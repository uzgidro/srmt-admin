package piezometers

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
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type PiezometerUpdater interface {
	UpdatePiezometer(ctx context.Context, id int64, req filtration.UpdatePiezometerRequest, userID int64) error
}

func Update(log *slog.Logger, updater PiezometerUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.filtration.piezometers.Update"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req filtration.UpdatePiezometerRequest
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

		if err := updater.UpdatePiezometer(r.Context(), id, req, userID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("piezometer not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Piezometer not found"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("piezometer name conflict", sl.Err(err))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Piezometer name already exists"))
				return
			}
			log.Error("failed to update piezometer", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update piezometer"))
			return
		}

		log.Info("piezometer updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
