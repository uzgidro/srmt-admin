package stock

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
)

type Saver interface {
	SaveStockData(ctx context.Context, jsonData string) error
}

func New(log *slog.Logger, saver Saver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sc.callback.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		rawJSON, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error("failed to read callback request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("could not read request body"))
			return
		}
		defer r.Body.Close()

		if len(rawJSON) == 0 {
			log.Warn("received empty request body")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("empty request body is not allowed"))
			return
		}

		jsonData := string(rawJSON)

		log.Info("received processed stock data, saving to storage")

		if err := saver.SaveStockData(r.Context(), jsonData); err != nil {
			log.Error("failed to save stock data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save data"))
			return
		}

		log.Info("stock data saved successfully")

		render.Status(r, http.StatusOK)
	}
}
