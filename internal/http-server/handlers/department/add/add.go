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
	Name           string  `json:"name" validate:"required"`
	Description    *string `json:"description,omitempty"`
	OrganizationID int64   `json:"organization_id" validate:"required"`
}

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

// DepartmentAdder - интерфейс, который должен реализовать репозиторий
type DepartmentAdder interface {
	AddDepartment(ctx context.Context, name string, description *string, orgID int64) (int64, error)
}

func New(log *slog.Logger, adder DepartmentAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.department.add.New"
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

		id, err := adder.AddDepartment(r.Context(), req.Name, req.Description, req.OrganizationID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("organization not found for department", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("department name duplicate", "name", req.Name)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Department with this name already exists"))
				return
			}
			log.Error("failed to add department", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add department"))
			return
		}

		log.Info("department added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: id})
	}
}
