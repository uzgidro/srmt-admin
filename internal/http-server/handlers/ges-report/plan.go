package gesreport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type PlanUpserter interface {
	BulkUpsertGESPlan(ctx context.Context, req model.BulkUpsertPlanRequest, userID int64) error
}

type PlanGetter interface {
	GetGESPlans(ctx context.Context, year int) ([]model.ProductionPlan, error)
}

func BulkUpsertPlan(log *slog.Logger, repo PlanUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.BulkUpsertPlan"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var req model.BulkUpsertPlanRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request format"))
			return
		}

		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		if err := repo.BulkUpsertGESPlan(r.Context(), req, userID); err != nil {
			log.Error("failed to bulk upsert ges plan", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save plans"))
			return
		}

		log.Info("ges plans bulk upserted", slog.Int("count", len(req.Plans)))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

func GetPlans(log *slog.Logger, repo PlanGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.GetPlans"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		yearStr := r.URL.Query().Get("year")
		if yearStr == "" {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("year is required"))
			return
		}
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("year must be a valid integer"))
			return
		}

		plans, err := repo.GetGESPlans(r.Context(), year)
		if err != nil {
			log.Error("failed to get ges plans", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve plans"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, plans)
	}
}
