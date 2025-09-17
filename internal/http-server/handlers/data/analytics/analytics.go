package analytics

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	complexValue "srmt-admin/internal/lib/model/dto/complex-value"
	"strconv"
)

type DataGetter interface {
	GetSelectedYearDataIncome(ctx context.Context, id, year int) (complexValue.ComplexValue, error)
}

func New(log *slog.Logger, dataGetter DataGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.data.analytics.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Получаем 'id' из URL
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Error("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid or missing 'id' parameter"))
			return
		}

		// 2. Получаем 'year' из query-параметров (ИСПРАВЛЕНО)
		yearStr := r.URL.Query().Get("year")
		year, err := strconv.Atoi(yearStr)
		if err != nil {
			log.Error("invalid 'year' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid or missing 'year' parameter"))
			return
		}

		income, err := dataGetter.GetSelectedYearDataIncome(r.Context(), id, year)
		if err != nil {
			log.Error("failed to get analytics data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Internal server error"))
			return
		}

		render.JSON(w, r, income)
	}
}
