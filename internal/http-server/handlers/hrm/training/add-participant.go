package training

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type ParticipantAdder interface {
	AddParticipant(ctx context.Context, trainingID, employeeID int64) (int64, error)
}

func AddParticipant(log *slog.Logger, svc ParticipantAdder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.AddParticipant"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		trainingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		var req dto.AddParticipantRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request body"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(err.Error()))
			return
		}

		id, err := svc.AddParticipant(r.Context(), trainingID, req.EmployeeID)
		if err != nil {
			if errors.Is(err, storage.ErrTrainingNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Training not found"))
				return
			}
			if errors.Is(err, storage.ErrTrainingFull) {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Training has reached maximum participants"))
				return
			}
			if errors.Is(err, storage.ErrAlreadyEnrolled) {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Employee is already enrolled in this training"))
				return
			}
			log.Error("failed to add participant", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add participant"))
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]int64{"id": id})
	}
}
