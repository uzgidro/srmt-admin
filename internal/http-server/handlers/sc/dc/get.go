package dc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type Getter interface {
	GetDC(ctx context.Context) (string, error)
}

func Get(log *slog.Logger, getter Getter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sc.dc.Get"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		jsonData, err := getter.GetDC(r.Context())
		if err != nil {
			if errors.Is(err, storage.ErrDataNotFound) {
				log.Warn("no DC data found in storage")
				w.WriteHeader(http.StatusNotFound)
				return
			}

			log.Error("failed to get DC data", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to get data"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jsonData))
	}
}
