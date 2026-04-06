package infraeventcategory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type createRequest struct {
	Slug        string `json:"slug" validate:"required"`
	DisplayName string `json:"display_name" validate:"required"`
	Label       string `json:"label" validate:"required"`
	SortOrder   int    `json:"sort_order"`
}

type createResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

type categoryCreator interface {
	CreateInfraEventCategory(ctx context.Context, slug, displayName, label string, sortOrder int) (int64, error)
}

func Create(log *slog.Logger, creator categoryCreator) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.infra-event-category.create"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req createRequest
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

		id, err := creator.CreateInfraEventCategory(r.Context(), req.Slug, req.DisplayName, req.Label, req.SortOrder)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate slug", slog.String("slug", req.Slug))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Category with this slug already exists"))
				return
			}
			log.Error("failed to create category", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create category"))
			return
		}

		log.Info("category created", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, createResponse{Response: resp.OK(), ID: id})
	}
}
