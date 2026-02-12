package salary

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

func GetPayslip(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusNotImplemented)
		render.JSON(w, r, stubResponse{Error: "not implemented yet"})
	}
}
