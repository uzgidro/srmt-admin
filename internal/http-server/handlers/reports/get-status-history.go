package reports

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/report"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type reportStatusHistoryGetter interface {
	GetReportStatusHistory(ctx context.Context, reportID int64) ([]report.StatusHistory, error)
}

func GetStatusHistory(log *slog.Logger, getter reportStatusHistoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reports.get-status-history"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		history, err := getter.GetReportStatusHistory(r.Context(), id)
		if err != nil {
			log.Error("failed to get status history", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve status history"))
			return
		}

		log.Info("successfully retrieved report status history", slog.Int64("id", id), slog.Int("count", len(history)))
		render.JSON(w, r, history)
	}
}
