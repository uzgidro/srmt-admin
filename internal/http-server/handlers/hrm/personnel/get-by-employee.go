package personnel

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/personnel"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ByEmployeeGetter interface {
	GetByEmployeeID(ctx context.Context, employeeID int64) (*personnel.Record, error)
}

func GetByEmployee(log *slog.Logger, svc ByEmployeeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.personnel.GetByEmployee"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		employeeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid employee ID"))
			return
		}

		rec, err := svc.GetByEmployeeID(r.Context(), employeeID)
		if err != nil {
			if errors.Is(err, storage.ErrPersonnelRecordNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Personnel record not found"))
				return
			}
			log.Error("failed to get personnel record by employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get record"))
			return
		}

		render.JSON(w, r, rec)
	}
}
