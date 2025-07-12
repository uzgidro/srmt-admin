package edit

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"strconv"
)

type Request struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type Response struct {
	resp.Response
}

type RoleEditor interface {
	EditRole(ctx context.Context, id int64, name, description string) error
}

func New(log *slog.Logger, editor RoleEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.role.edit.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		roleID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid role ID", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid role id"))
			return
		}

		var req Request

		// Decode JSON
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request"))
			return
		}

		log.Info("request parsed", slog.Any("req", req))

		var newName, newDescription string
		if req.Name != nil {
			newName = *req.Name
		}
		if req.Description != nil {
			newDescription = *req.Description
		}

		// edit role
		err = editor.EditRole(r.Context(), roleID, newName, newDescription)
		if err != nil {
			log.Info("failed to edit role", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to add role"))
			return
		}

		log.Info("role successfully edited", slog.Int64("id", roleID))

		render.Status(r, http.StatusOK)
	}
}
