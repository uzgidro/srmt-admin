package analytics

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/dto/analytics"
	complexValue "srmt-admin/internal/lib/model/dto/complex-value"
	"srmt-admin/internal/storage"
	"strconv"
	"time"
)

type DataGetter interface {
	GetSelectedYearDataIncome(ctx context.Context, id, year int) (complexValue.Model, error)
	GetDataByYears(ctx context.Context, id int) (complexValue.Model, error)
	GetAvgData(ctx context.Context, id int) (complexValue.Model, error)
	GetTenYearsAvgData(ctx context.Context, id int) (complexValue.Model, error)
	GetExtremumYear(ctx context.Context, id int, extremumType string) (int, error)
}

func New(log *slog.Logger, dataGetter DataGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.data.analytics.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Error("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid or missing 'id' parameter"))
			return
		}

		result := analytics.Model{}

		yearsData, err := dataGetter.GetDataByYears(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("no data found for reservoir", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound(""))
				return
			}
			log.Error("failed to get data by years", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to get yearly data"))
			return
		}
		result.ReservoirID = yearsData.ReservoirID
		result.Reservoir = yearsData.Reservoir
		result.Years = yearsData.Data

		year := time.Now().Year()

		currentYearData, err := dataGetter.GetSelectedYearDataIncome(r.Context(), id, year)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			log.Warn("failed to get current year data", sl.Err(err), slog.Int("year", year))
		}
		if currentYearData.Data != nil {
			result.CurrentYear = currentYearData.Data
		}

		pastYearData, err := dataGetter.GetSelectedYearDataIncome(r.Context(), id, year-1)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			log.Warn("failed to get past year data", sl.Err(err), slog.Int("year", year-1))
		}
		if pastYearData.Data != nil {
			result.PastYears = pastYearData.Data
		}

		avg, err := dataGetter.GetAvgData(r.Context(), id)
		if err != nil {
			log.Warn("failed to get avg data", sl.Err(err))
		}
		if avg.Data != nil {
			result.Avg = avg.Data
		}

		tenAvg, err := dataGetter.GetTenYearsAvgData(r.Context(), id)
		if err != nil {
			log.Warn("failed to get ten years avg data", sl.Err(err))
		}
		if tenAvg.Data != nil {
			result.TenAvg = tenAvg.Data
		}

		maxYear, err := dataGetter.GetExtremumYear(r.Context(), id, "max")
		if err != nil {
			log.Warn("failed to get max year", sl.Err(err))
		}
		maxIncome, err := dataGetter.GetSelectedYearDataIncome(r.Context(), id, maxYear)
		if err != nil {
			log.Warn("failed to get max income", sl.Err(err))
		}
		if maxIncome.Data != nil {
			result.Max = maxIncome.Data
		}

		minYear, err := dataGetter.GetExtremumYear(r.Context(), id, "min")
		if err != nil {
			log.Warn("failed to get min year", sl.Err(err))
		}
		minIncome, err := dataGetter.GetSelectedYearDataIncome(r.Context(), id, minYear)
		if err != nil {
			log.Warn("failed to get min income", sl.Err(err))
		}
		if minIncome.Data != nil {
			result.Min = minIncome.Data
		}

		render.JSON(w, r, result)
	}
}
