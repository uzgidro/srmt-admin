package get_by_id

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/contact"
	"srmt-admin/internal/storage"
	"strconv"
)

// ContactGetter - интерфейс для получения одного
type ContactGetter interface {
	GetContactByID(ctx context.Context, id int64) (*contact.Model, error)
}

func New(log *slog.Logger, getter ContactGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.get_by_id.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		contact, err := getter.GetContactByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("contact not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Contact not found"))
				return
			}
			log.Error("failed to get contact", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve contact"))
			return
		}

		log.Info("successfully retrieved contact", slog.Int64("id", contact.ID))
		render.JSON(w, r, contact)
	}
}
