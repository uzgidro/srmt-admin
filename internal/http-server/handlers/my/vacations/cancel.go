package vacations

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	vacationmodel "srmt-admin/internal/lib/model/hrm/vacation"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type VacationGetter interface {
	GetByID(ctx context.Context, id int64) (*vacationmodel.Vacation, error)
}

type Canceller interface {
	Cancel(ctx context.Context, id int64) error
}

type CancelService interface {
	VacationGetter
	Canceller
}

func Cancel(log *slog.Logger, svc CancelService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.vacations.Cancel"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		// Verify ownership
		vac, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrVacationNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			log.Error("failed to get vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get vacation"))
			return
		}

		if vac.EmployeeID != claims.ContactID {
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("You can only cancel your own vacations"))
			return
		}

		if err := svc.Cancel(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrInvalidStatus) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Only draft, pending, or approved vacations can be cancelled"))
				return
			}
			log.Error("failed to cancel vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to cancel vacation"))
			return
		}

		render.JSON(w, r, resp.OK())
	}
}
