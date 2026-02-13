package recruiting

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func GetStats(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetStats"
		_ = log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		render.Status(r, http.StatusNotImplemented)
		render.JSON(w, r, map[string]string{"error": "Recruiting stats not implemented yet"})
	}
}
