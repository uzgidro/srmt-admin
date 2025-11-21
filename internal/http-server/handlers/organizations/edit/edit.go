package edit

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
	"srmt-admin/internal/storage"
	"strconv"
)

type Request struct {
	Name                 *string `json:"name,omitempty"`
	ParentOrganizationID **int64 `json:"parent_organization_id,omitempty"`
	TypeIDs              []int64 `json:"type_ids,omitempty"`
}

type OrganizationEditor interface {
	EditOrganization(ctx context.Context, id int64, name *string, parentID **int64, typeIDs []int64) error
}

func New(log *slog.Logger, editor OrganizationEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.organizations.patch.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		orgID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid organization ID format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid organization ID"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = editor.EditOrganization(r.Context(), orgID, req.Name, req.ParentOrganizationID, req.TypeIDs)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("organization not found", "id", orgID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Organization not found"))
				return
			}
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("organization name conflict")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Organization with this name already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("parent organization or type not found")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Parent organization or one of the types not found"))
				return
			}
			log.Error("failed to edit organization", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to edit organization"))
			return
		}

		log.Info("organization updated successfully", slog.Int64("id", orgID))
		render.Status(r, http.StatusNoContent)
	}
}
