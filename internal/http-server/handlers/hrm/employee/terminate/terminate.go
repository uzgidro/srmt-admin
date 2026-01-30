package terminate

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
)

// Request represents the termination request
type Request struct {
	TerminationDate time.Time `json:"termination_date" validate:"required"`
	Reason          *string   `json:"reason,omitempty"`
}

// EmployeeTerminator defines the interface for terminating employees
type EmployeeTerminator interface {
	TerminateEmployee(ctx context.Context, id int64, terminationDate time.Time) error
}

// New creates a new terminate employee handler
func New(log *slog.Logger, terminator EmployeeTerminator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.employee.terminate.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

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

		err = terminator.TerminateEmployee(r.Context(), id, req.TerminationDate)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("employee not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Employee not found"))
				return
			}
			log.Error("failed to terminate employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to terminate employee"))
			return
		}

		log.Info("employee terminated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}
