package personnel

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Creator interface {
	Create(ctx context.Context, req dto.CreatePersonnelRecordRequest) (int64, error)
}

type CreateResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

func Create(log *slog.Logger, svc Creator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.personnel.Create"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req dto.CreatePersonnelRecordRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := svc.Create(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) || errors.Is(err, storage.ErrDuplicate) {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Personnel record for this employee already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee, department, or position ID"))
				return
			}
			log.Error("failed to create personnel record", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create record"))
			return
		}

		log.Info("personnel record created", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, CreateResponse{Response: resp.OK(), ID: id})
	}
}
