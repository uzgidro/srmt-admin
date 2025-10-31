package edit

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
)

type Request struct {
	Name string `json:"name" validate:"required"`
}

type OrganizationTypeEditor interface {
	EditOrganizationType(ctx context.Context, id string, name string) error
}

func New(log *slog.Logger, editor OrganizationTypeEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organization-types.edit.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id := chi.URLParam(r, "id")
		if id == "" {
			log.Warn("id is empty")
			render.JSON(w, r, resp.BadRequest("invalid request"))
			return
		}

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, resp.BadRequest("failed to decode request"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			log.Error("invalid request", sl.Err(err))
			render.JSON(w, r, resp.BadRequest("invalid request"))
			return
		}

		err = editor.EditOrganizationType(r.Context(), id, req.Name)
		if err != nil {
			log.Error("failed to edit organization type", sl.Err(err))
			render.JSON(w, r, resp.InternalServerError("failed to edit organization type"))
			return
		}

		log.Info("organization type edited", slog.String("id", id))

		render.JSON(w, r, resp.OK())
	}
}
