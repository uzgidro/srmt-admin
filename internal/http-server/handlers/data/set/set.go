package set

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/data"
	dataConvert "srmt-admin/internal/lib/service/data-convert"
	"strconv"
	"time"
)

type Request struct {
	Current    float64   `json:"current" validate:"required"`
	Resistance float64   `json:"resistance" validate:"required"`
	Time       time.Time `json:"time" validate:"required"`
}

type DataSaver interface {
	GetIndicator(ctx context.Context, resID int64) (float64, error)
	SaveData(ctx context.Context, data data.Model) error
}

func New(log *slog.Logger, saver DataSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.data.andijan.set.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		resID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid role ID", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid role id"))
			return
		}

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		log.Info("request parsed", slog.Any("req", req))

		if err := validator.New().Struct(req); err != nil {
			var validationErrors validator.ValidationErrors
			errors.As(err, &validationErrors)

			log.Error("failed to validate request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to validate request"))
			return
		}

		indicatorLevel, err := saver.GetIndicator(r.Context(), resID)
		if err != nil {
			log.Error("failed to get indicator level", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get indicator level"))
			return
		}

		data, err := dataConvert.Convert(resID, indicatorLevel, req.Current, req.Resistance)
		data.Time = req.Time

		if err = saver.SaveData(r.Context(), data); err != nil {
			log.Error("failed to save data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save data"))
			return
		}

		log.Info("successfully saved data")

		render.Status(r, http.StatusCreated)
	}
}
