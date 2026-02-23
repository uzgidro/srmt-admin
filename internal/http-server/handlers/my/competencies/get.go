package competencies

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/hrm/competency"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type CompetencyGetter interface {
	GetEmployeeAssessments(ctx context.Context, employeeID int64) ([]*competency.AssessmentSession, error)
}

func Get(log *slog.Logger, svc CompetencyGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.my.competencies.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("unauthorized"))
			return
		}

		assessments, err := svc.GetEmployeeAssessments(r.Context(), claims.ContactID)
		if err != nil {
			log.Error("failed to get assessments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get competency data"))
			return
		}

		if assessments == nil {
			assessments = []*competency.AssessmentSession{}
		}

		render.JSON(w, r, assessments)
	}
}
