package competencies

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

type stubResponse struct {
	Error string `json:"error"`
}

func Get(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusNotImplemented)
		render.JSON(w, r, stubResponse{Error: "not implemented yet"})
	}
}
