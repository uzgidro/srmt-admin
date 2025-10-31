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

type Request struct {
	Name                 string  `json:"name" validate:"required"`
	ParentOrganizationID *int64  `json:"parent_organization_id,omitempty"`
	TypeIDs              []int64 `json:"type_ids" validate:"required,min=1"`
}

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

type OrganizationAdder interface {
	AddOrganization(ctx context.Context, name string, parentID *int64, typeIDs []int64) (int64, error)
}

func New(log *slog.Logger, adder OrganizationAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organizations.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(validationErrors))
			return
		}

		id, err := adder.AddOrganization(r.Context(), req.Name, req.ParentOrganizationID, req.TypeIDs)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("organization already exists", "name", req.Name)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Organization with this name already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("parent organization or type not found", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Parent organization or one of the types not found"))
				return
			}
			log.Error("failed to add organization", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add organization"))
			return
		}

		log.Info("organization added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.Ok(), ID: id})
	}
}
