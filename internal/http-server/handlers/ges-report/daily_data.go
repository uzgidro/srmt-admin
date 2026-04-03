package gesreport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	model "srmt-admin/internal/lib/model/ges-report"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type DailyDataUpserter interface {
	UpsertGESDailyData(ctx context.Context, req model.UpsertDailyDataRequest, userID int64) error
}

type DailyDataGetter interface {
	GetGESDailyData(ctx context.Context, organizationID int64, date string) (*model.DailyData, error)
}

func UpsertDailyData(log *slog.Logger, repo DailyDataUpserter) http.HandlerFunc {
	validate := validator.New()
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.UpsertDailyData"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("not authenticated"))
			return
		}

		var req model.UpsertDailyDataRequest
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

		if _, err := time.Parse("2006-01-02", req.Date); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}

		if err := repo.UpsertGESDailyData(r.Context(), req, userID); err != nil {
			log.Error("failed to upsert ges daily data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save daily data"))
			return
		}

		log.Info("ges daily data upserted",
			slog.Int64("organization_id", req.OrganizationID),
			slog.String("date", req.Date),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, resp.OK())
	}
}

func GetDailyData(log *slog.Logger, repo DailyDataGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.ges-report.GetDailyData"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgID, err := parseIntParam(r, "organization_id")
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_id is required"))
			return
		}

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

		data, err := repo.GetGESDailyData(r.Context(), orgID, date)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusOK)
				render.JSON(w, r, nil)
				return
			}
			log.Error("failed to get ges daily data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve daily data"))
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, data)
	}
}
