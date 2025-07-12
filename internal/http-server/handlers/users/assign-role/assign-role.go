package assign_role

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Request struct {
	RoleID int64 `json:"role_id" validate:"required"`
}

type RoleAssigner interface {
	AssignRole(ctx context.Context, userID, roleID int64) error
}

func New(log *slog.Logger, roleAssigner RoleAssigner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.users.assign-role.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
		if err != nil {
			log.Warn("invalid user ID", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid user id"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid request"))
			return
		}

		err = roleAssigner.AssignRole(r.Context(), userID, req.RoleID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("user or role not found", "user_id", userID, "role_id", req.RoleID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.NotFound("user or role not found"))
				return
			}
			log.Error("failed to assign role", "error", err)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}

		log.Info("role assigned successfully", "user_id", userID, "role_id", req.RoleID)

		render.Status(r, http.StatusNoContent)
	}
}
