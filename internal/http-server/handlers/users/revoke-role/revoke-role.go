package revoke_role

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type RoleRevoker interface {
	RevokeRole(ctx context.Context, userID, roleID int64) error
}

func New(log *slog.Logger, roleRevoker RoleRevoker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := log.With(slog.String("op", "handlers.users.revoke_role.New"))

		userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
		if err != nil {
			log.Warn("invalid user ID", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid user id"))
			return
		}

		roleID, err := strconv.ParseInt(chi.URLParam(r, "roleID"), 10, 64)
		if err != nil {
			log.Warn("invalid role ID", "error", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid role id"))
			return
		}

		err = roleRevoker.RevokeRole(r.Context(), userID, roleID)
		if err != nil {
			log.Error("failed to revoke role", "error", err)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}

		log.Info("role revoked successfully", "user_id", userID, "role_id", roleID)

		render.Status(r, http.StatusNoContent)
	}
}
