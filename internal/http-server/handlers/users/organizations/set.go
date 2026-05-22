// Package organizations provides the admin endpoint for replacing a user's
// organization bindings (the user_organizations M2M table).
package organizations

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// Request is the body of PUT /users/{userID}/organizations.
//
// OrganizationIDs is required (a missing/null field is rejected) but an empty
// list is valid and clears all of the user's organization bindings.
type Request struct {
	OrganizationIDs []int64 `json:"organization_ids" validate:"required"`
}

// OrganizationSetter replaces a user's full organization list.
type OrganizationSetter interface {
	SetUserOrganizations(ctx context.Context, userID int64, orgIDs []int64) error
}

// New returns the handler for PUT /users/{userID}/organizations.
func New(log *slog.Logger, setter OrganizationSetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.users.organizations.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
		if err != nil {
			log.Warn("invalid user id", "error", err)
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
		if req.OrganizationIDs == nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("organization_ids is required"))
			return
		}

		if err := setter.SetUserOrganizations(r.Context(), userID, req.OrganizationIDs); err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("user or organization not found",
					"user_id", userID, "organization_ids", req.OrganizationIDs)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("user or organization not found"))
				return
			}
			log.Error("failed to set user organizations", "error", err)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}

		log.Info("user organizations set",
			"user_id", userID, "organization_ids", req.OrganizationIDs)

		// render.NoContent writes the 204 header directly — render.Status alone
		// only takes effect when a body is rendered, and a 204 has none.
		render.NoContent(w, r)
	}
}
