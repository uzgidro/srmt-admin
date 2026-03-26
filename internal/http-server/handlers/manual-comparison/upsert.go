package manualcomparison

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	manualcomparison "srmt-admin/internal/lib/model/manual-comparison"
	"srmt-admin/internal/lib/service/auth"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type ManualComparisonUpserter interface {
	UpsertManualComparison(ctx context.Context, req manualcomparison.UpsertRequest) error
}

type upsertRequest struct {
	OrganizationID       int64                        `json:"organization_id" validate:"required"`
	Date                 string                       `json:"date" validate:"required"`
	HistoricalFilterDate string                       `json:"historical_filter_date"`
	HistoricalPiezoDate  string                       `json:"historical_piezo_date"`
	Filters              []manualcomparison.FilterInput `json:"filters"`
	Piezos               []manualcomparison.PiezoInput  `json:"piezos"`
}

func Upsert(log *slog.Logger, upserter ManualComparisonUpserter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.manual-comparison.Upsert"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req upsertRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		validate := validator.New()
		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		if err := auth.CheckOrgAccess(r.Context(), req.OrganizationID); err != nil {
			log.Warn("access denied to organization", slog.Int64("org_id", req.OrganizationID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		if _, err := time.Parse("2006-01-02", req.Date); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'date' format (expected YYYY-MM-DD)"))
			return
		}

		if len(req.Filters) == 0 && len(req.Piezos) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("At least one filter or piezo measurement must be provided"))
			return
		}

		if err := upserter.UpsertManualComparison(r.Context(), manualcomparison.UpsertRequest{
			OrganizationID:       req.OrganizationID,
			Date:                 req.Date,
			HistoricalFilterDate: req.HistoricalFilterDate,
			HistoricalPiezoDate:  req.HistoricalPiezoDate,
			Filters:              req.Filters,
			Piezos:               req.Piezos,
			UserID:               userID,
		}); err != nil {
			log.Error("failed to upsert manual comparison", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to save manual comparison"))
			return
		}

		log.Info("manual comparison upserted",
			slog.String("date", req.Date),
			slog.Int64("org_id", req.OrganizationID),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}
