package get

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
)

// TransferGetter defines the interface for getting transfers
type TransferGetter interface {
	GetTransfers(ctx context.Context, filter hrm.TransferFilter) ([]*hrmmodel.Transfer, error)
}

// New creates a new get transfers handler
func New(log *slog.Logger, getter TransferGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.transfer.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.TransferFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'employee_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'employee_id' parameter"))
				return
			}
			filter.EmployeeID = &val
		}

		if transferType := q.Get("transfer_type"); transferType != "" {
			filter.TransferType = &transferType
		}

		if fromDateStr := q.Get("from_date"); fromDateStr != "" {
			val, err := time.Parse(time.DateOnly, fromDateStr)
			if err != nil {
				log.Warn("invalid 'from_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'from_date' parameter, use YYYY-MM-DD"))
				return
			}
			filter.FromDate = &val
		}

		if toDateStr := q.Get("to_date"); toDateStr != "" {
			val, err := time.Parse(time.DateOnly, toDateStr)
			if err != nil {
				log.Warn("invalid 'to_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'to_date' parameter, use YYYY-MM-DD"))
				return
			}
			filter.ToDate = &val
		}

		if deptIDStr := q.Get("department_id"); deptIDStr != "" {
			val, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'department_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'department_id' parameter"))
				return
			}
			filter.DepartmentID = &val
		}

		if orgIDStr := q.Get("organization_id"); orgIDStr != "" {
			val, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'organization_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
				return
			}
			filter.OrganizationID = &val
		}

		transfers, err := getter.GetTransfers(r.Context(), filter)
		if err != nil {
			log.Error("failed to get transfers", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve transfers"))
			return
		}

		log.Info("successfully retrieved transfers", slog.Int("count", len(transfers)))
		render.JSON(w, r, transfers)
	}
}
