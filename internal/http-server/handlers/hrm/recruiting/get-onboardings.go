package recruiting

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func GetOnboardings(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetOnboardings"
		_ = log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		render.JSON(w, r, []struct{}{})
	}
}
