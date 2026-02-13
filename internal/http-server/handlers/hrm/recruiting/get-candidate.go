package recruiting

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	recruiting "srmt-admin/internal/lib/model/hrm/recruiting"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type CandidateByIDGetter interface {
	GetCandidateByID(ctx context.Context, id int64) (*recruiting.Candidate, error)
}

func GetCandidate(log *slog.Logger, svc CandidateByIDGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetCandidate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		candidate, err := svc.GetCandidateByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrCandidateNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Candidate not found"))
				return
			}
			log.Error("failed to get candidate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve candidate"))
			return
		}

		render.JSON(w, r, candidate)
	}
}
