package recruiting

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type VacancyCloser interface {
	CloseVacancy(ctx context.Context, id int64) error
}

func CloseVacancy(log *slog.Logger, svc VacancyCloser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.CloseVacancy"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		if err := svc.CloseVacancy(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrVacancyNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacancy not found"))
				return
			}
			if errors.Is(err, storage.ErrVacancyNotPublished) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Only published vacancies can be closed"))
				return
			}
			log.Error("failed to close vacancy", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to close vacancy"))
			return
		}

		render.JSON(w, r, resp.OK())
	}
}
