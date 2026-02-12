package personnel

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/personnel"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Getter interface {
	GetByID(ctx context.Context, id int64) (*personnel.Record, error)
}

func GetByID(log *slog.Logger, svc Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.personnel.GetByID"
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

		rec, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrPersonnelRecordNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Personnel record not found"))
				return
			}
			log.Error("failed to get personnel record", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get record"))
			return
		}

		// hrm_employee can only see own record
		if hasOnlyRole(claims.Roles, "hrm_employee") && rec.EmployeeID != claims.ContactID {
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		render.JSON(w, r, rec)
	}
}
