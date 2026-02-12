package personnel

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/personnel"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Lister interface {
	GetAll(ctx context.Context, filters dto.PersonnelRecordFilters) ([]*personnel.Record, error)
	GetByEmployeeID(ctx context.Context, employeeID int64) (*personnel.Record, error)
}

func GetAll(log *slog.Logger, svc Lister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.personnel.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		// hrm_employee can only see own record
		isEmployee := hasOnlyRole(claims.Roles, "hrm_employee")
		if isEmployee {
			rec, err := svc.GetByEmployeeID(r.Context(), claims.ContactID)
			if err != nil {
				log.Error("failed to get own personnel record", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to get record"))
				return
			}
			if rec == nil {
				render.JSON(w, r, []*personnel.Record{})
				return
			}
			render.JSON(w, r, []*personnel.Record{rec})
			return
		}

		var filters dto.PersonnelRecordFilters
		q := r.URL.Query()
		if v := q.Get("department_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.DepartmentID = &val
		}
		if v := q.Get("position_id"); v != "" {
			val, _ := strconv.ParseInt(v, 10, 64)
			filters.PositionID = &val
		}
		if v := q.Get("status"); v != "" {
			filters.Status = &v
		}
		if v := q.Get("search"); v != "" {
			filters.Search = &v
		}

		records, err := svc.GetAll(r.Context(), filters)
		if err != nil {
			log.Error("failed to get personnel records", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve records"))
			return
		}

		if records == nil {
			records = []*personnel.Record{}
		}
		render.JSON(w, r, records)
	}
}

func hasOnlyRole(roles []string, role string) bool {
	hrmRoles := map[string]bool{"hrm_admin": true, "hrm_manager": true, "hrm_employee": true}
	hasTarget := false
	for _, r := range roles {
		if r == role {
			hasTarget = true
		}
		if hrmRoles[r] && r != role {
			return false // has a higher HRM role
		}
	}
	return hasTarget
}
