package gesreport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	gesreportservice "srmt-admin/internal/lib/service/ges-report"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ReportBuilder interface {
	BuildDailyReport(ctx context.Context, date string, cascadeOrgID *int64) (*model.DailyReport, error)
}

// consumptionViolationsToDetails converts the typed service-side violation
// slice into the open-ended []resp.Detail shape the structured error helper
// expects. Each entry mirrors the field names from the plan's API contract
// (organization_id, organization_name, date, idle_m3_s, consumption_m3_s).
func consumptionViolationsToDetails(vs []gesreportservice.ConsumptionViolation) []resp.Detail {
	out := make([]resp.Detail, 0, len(vs))
	for _, v := range vs {
		out = append(out, resp.Detail{
			"organization_id":   v.OrganizationID,
			"organization_name": v.OrganizationName,
			"date":              v.Date,
			"idle_m3_s":         v.IdleM3s,
			"consumption_m3_s":  v.ConsumptionM3s,
		})
	}
	return out
}

func GetReport(log *slog.Logger, svc ReportBuilder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.GetReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		date := r.URL.Query().Get("date")
		if date == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("date is required (YYYY-MM-DD)"))
			return
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}

		// Determine cascade scope: if user is NOT sc/rais, restrict to their cascade org.
		var cascadeOrgID *int64
		if claims, ok := mwauth.ClaimsFromContext(r.Context()); ok && claims != nil {
			isSuperUser := false
			for _, role := range claims.Roles {
				if role == "sc" || role == "rais" {
					isSuperUser = true
					break
				}
			}
			if !isSuperUser {
				// A non-admin (cascade role) without any organization is a
				// broken account — deny rather than fall through to the
				// unscoped full report.
				if len(claims.OrganizationIDs) == 0 {
					log.Warn("non-admin caller without organization id")
					render.Status(r, http.StatusForbidden)
					render.JSON(w, r, resp.Forbidden("user has no organization assigned"))
					return
				}
				// ges-report cascade-scope ограничен первым каскадом
				// пользователя; multi-cascade отчёт — отдельная задача.
				id := claims.OrganizationIDs[0]
				cascadeOrgID = &id
			}
		}

		report, err := svc.BuildDailyReport(r.Context(), date, cascadeOrgID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("report data not found"))
				return
			}
			var rve *gesreportservice.ReportValidationError
			if errors.As(err, &rve) {
				log.Warn("report validation rejected build",
					slog.String("code", rve.Code),
					slog.Int("violations", len(rve.Violations)))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequestStructured(
					rve.Code,
					rve.Error(),
					consumptionViolationsToDetails(rve.Violations),
				))
				return
			}
			log.Error("failed to build daily report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to build report"))
			return
		}

		log.Info("ges daily report built", slog.String("date", date))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, report)
	}
}
