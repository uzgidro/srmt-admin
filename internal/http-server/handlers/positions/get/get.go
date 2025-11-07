package get

import (
	"context"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/position"
)

type PositionGetter interface {
	GetAllPositions(ctx context.Context) ([]*position.Model, error)
}

func New(log *slog.Logger, getter PositionGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.position.get_all.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// 1. Вызываем метод репозитория (фильтров нет)
		positions, err := getter.GetAllPositions(r.Context())
		if err != nil {
			log.Error("failed to get all positions", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve positions"))
			return
		}

		log.Info("successfully retrieved positions", slog.Int("count", len(positions)))
		render.JSON(w, r, positions)
	}
}
