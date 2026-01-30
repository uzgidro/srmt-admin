package get

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
)

// EmployeeGetter defines the interface for getting employees
type EmployeeGetter interface {
	GetAllEmployees(ctx context.Context, filter hrm.EmployeeFilter) ([]*hrmmodel.Employee, error)
}

// New creates a new get employees handler
func New(log *slog.Logger, getter EmployeeGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.employee.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.EmployeeFilter
		q := r.URL.Query()

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

		if posIDStr := q.Get("position_id"); posIDStr != "" {
			val, err := strconv.ParseInt(posIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'position_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'position_id' parameter"))
				return
			}
			filter.PositionID = &val
		}

		if managerIDStr := q.Get("manager_id"); managerIDStr != "" {
			val, err := strconv.ParseInt(managerIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'manager_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'manager_id' parameter"))
				return
			}
			filter.ManagerID = &val
		}

		if empType := q.Get("employment_type"); empType != "" {
			filter.EmploymentType = &empType
		}

		if empStatus := q.Get("employment_status"); empStatus != "" {
			filter.EmploymentStatus = &empStatus
		}

		if search := q.Get("search"); search != "" {
			filter.Search = &search
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, err := strconv.Atoi(limitStr)
			if err != nil {
				log.Warn("invalid 'limit' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'limit' parameter"))
				return
			}
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, err := strconv.Atoi(offsetStr)
			if err != nil {
				log.Warn("invalid 'offset' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'offset' parameter"))
				return
			}
			filter.Offset = val
		}

		employees, err := getter.GetAllEmployees(r.Context(), filter)
		if err != nil {
			log.Error("failed to get employees", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employees"))
			return
		}

		log.Info("successfully retrieved employees", slog.Int("count", len(employees)))
		render.JSON(w, r, employees)
	}
}
