package gessummary

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	gesproduction "srmt-admin/internal/lib/model/ges-production"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	Date                  string  `json:"date"`
	TotalEnergyProduction float64 `json:"total_energy_production"`
}

type Saver interface {
	UpsertGesProduction(ctx context.Context, data gesproduction.Model) error
}

func New(log *slog.Logger, saver Saver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sc.callback.ges-summary.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request body"))
			return
		}

		// Validate date format
		if _, err := time.Parse("2006-01-02", req.Date); err != nil {
			log.Error("invalid date format", sl.Err(err), slog.String("date", req.Date))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}

		model := gesproduction.Model{
			Date:                  req.Date,
			TotalEnergyProduction: req.TotalEnergyProduction,
		}

		if err := saver.UpsertGesProduction(r.Context(), model); err != nil {
			log.Error("failed to save ges production", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save data"))
			return
		}

		log.Info("ges production saved successfully", slog.String("date", req.Date), slog.Float64("value", req.TotalEnergyProduction))
		render.JSON(w, r, resp.OK())
	}
}
