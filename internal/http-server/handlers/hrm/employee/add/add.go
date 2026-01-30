package add

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
)

// EmployeeAdder defines the interface for adding employees
type EmployeeAdder interface {
	AddEmployee(ctx context.Context, req hrm.AddEmployeeRequest) (int64, error)
}

// Response represents the add employee response
type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

// New creates a new add employee handler
func New(log *slog.Logger, adder EmployeeAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.employee.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddEmployeeRequest
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

		id, err := adder.AddEmployee(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "contact_id", req.ContactID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid contact_id, user_id, or manager_id"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate employee", "contact_id", req.ContactID)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Employee with this contact already exists"))
				return
			}
			log.Error("failed to add employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add employee"))
			return
		}

		log.Info("employee added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: id})
	}
}
