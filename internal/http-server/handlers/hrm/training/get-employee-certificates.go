package training

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/training"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type EmployeeCertificatesGetter interface {
	GetEmployeeCertificates(ctx context.Context, employeeID int64) ([]*training.Certificate, error)
}

func GetEmployeeCertificates(log *slog.Logger, svc EmployeeCertificatesGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetEmployeeCertificates"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		employeeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid ID"))
			return
		}

		certs, err := svc.GetEmployeeCertificates(r.Context(), employeeID)
		if err != nil {
			log.Error("failed to get employee certificates", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve certificates"))
			return
		}

		render.JSON(w, r, certs)
	}
}
