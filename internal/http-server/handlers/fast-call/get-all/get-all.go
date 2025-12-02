package get_all

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/fast_call"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type fastCallGetter interface {
	GetAllFastCalls(ctx context.Context) ([]*fast_call.Model, error)
}

func New(log *slog.Logger, getter fastCallGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.fast_call.get_all.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		fastCalls, err := getter.GetAllFastCalls(r.Context())
		if err != nil {
			log.Error("failed to get fast calls", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve fast calls"))
			return
		}

		log.Info("successfully retrieved fast calls", slog.Int("count", len(fastCalls)))
		render.JSON(w, r, fastCalls)
	}
}
