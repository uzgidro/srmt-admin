package infraeventcategory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type updateRequest struct {
	Slug        string `json:"slug" validate:"required"`
	DisplayName string `json:"display_name" validate:"required"`
	Label       string `json:"label" validate:"required"`
	SortOrder   int    `json:"sort_order"`
}

type categoryUpdater interface {
	UpdateInfraEventCategory(ctx context.Context, id int64, slug, displayName, label string, sortOrder int) error
}

func Update(log *slog.Logger, updater categoryUpdater) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event-category.update"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req updateRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		err = updater.UpdateInfraEventCategory(r.Context(), id, req.Slug, req.DisplayName, req.Label, req.SortOrder)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Category not found"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Category with this slug already exists"))
				return
			}
			log.Error("failed to update category", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update category"))
			return
		}

		log.Info("category updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
