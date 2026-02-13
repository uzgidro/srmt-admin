package timesheet

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

type HolidayCreator interface {
	CreateHoliday(ctx context.Context, req dto.CreateHolidayRequest) (int64, error)
}

type CreateHolidayResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

func CreateHoliday(log *slog.Logger, svc HolidayCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.CreateHoliday"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req dto.CreateHolidayRequest
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

		id, err := svc.CreateHoliday(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrHolidayAlreadyExists) {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Holiday already exists for this date"))
				return
			}
			log.Error("failed to create holiday", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create holiday"))
			return
		}

		log.Info("holiday created", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, CreateHolidayResponse{Response: resp.OK(), ID: id})
	}
}
