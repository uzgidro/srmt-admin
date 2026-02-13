package recruiting

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type CandidateCreator interface {
	CreateCandidate(ctx context.Context, req dto.CreateCandidateRequest) (int64, error)
}

func CreateCandidate(log *slog.Logger, svc CandidateCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.CreateCandidate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req dto.CreateCandidateRequest
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

		id, err := svc.CreateCandidate(r.Context(), req)
		if err != nil {
			log.Error("failed to create candidate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create candidate"))
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]int64{"id": id})
	}
}
