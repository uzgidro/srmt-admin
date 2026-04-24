package shutdowns

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"strconv"
	"time"
)

// Request (JSON DTO)
type editRequest struct {
	OrganizationID      *int64     `json:"organization_id,omitempty"`
	StartTime           *time.Time `json:"start_time,omitempty"`
	EndTime             *time.Time `json:"end_time,omitempty"`
	Reason              *string    `json:"reason,omitempty"`
	GenerationLossMwh   *float64   `json:"generation_loss,omitempty"`
	ReportedByContactID *int64     `json:"reported_by_contact_id,omitempty"`

	IdleDischargeVolume *float64 `json:"idle_discharge_volume,omitempty"`
	FileIDs             []int64  `json:"file_ids,omitempty"`
}

type shutdownEditor interface {
	EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error
	UnlinkShutdownFiles(ctx context.Context, shutdownID int64) error
	LinkShutdownFiles(ctx context.Context, shutdownID int64, fileIDs []int64) error
	GetShutdownOrganizationID(ctx context.Context, id int64) (int64, error)
	GetShutdownCreatedByUserID(ctx context.Context, id int64) (sql.NullInt64, error)
	GetOrganizationParentID(ctx context.Context, orgID int64) (*int64, error)
}

func Edit(log *slog.Logger, editor shutdownEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Lookup current org to run RBAC check against it before mutating.
		curOrgID, err := editor.GetShutdownOrganizationID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("failed to load shutdown org", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to load shutdown"))
			return
		}

		// Access check on current org — foreign resource → 404 (enumeration defense).
		if err := auth.CheckCascadeStationAccess(r.Context(), curOrgID, editor); err != nil {
			if errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrNoOrganization) {
				log.Warn("cascade access denied on edit", slog.Int64("user_id", userID), slog.Int64("shutdown_id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("cascade access check failed", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
			return
		}

		// Ownership check — cascade callers may only edit records they created themselves.
		if err := auth.CheckShutdownOwnership(r.Context(), id, editor); err != nil {
			if errors.Is(err, auth.ErrForbidden) {
				log.Warn("cascade caller is not the owner of this shutdown",
					slog.Int64("user_id", userID),
					slog.Int64("shutdown_id", id),
				)
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("only the creator can edit this record"))
				return
			}
			if errors.Is(err, storage.ErrNotFound) {
				// Race: shutdown was deleted between GetShutdownOrganizationID and now.
				log.Warn("shutdown not found during ownership check", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			log.Error("ownership check failed", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to verify ownership"))
			return
		}

		// If caller is moving the record to a different org, also check access to the new org.
		if req.OrganizationID != nil {
			if err := auth.CheckCascadeStationAccess(r.Context(), *req.OrganizationID, editor); err != nil {
				if errors.Is(err, auth.ErrForbidden) || errors.Is(err, auth.ErrNoOrganization) {
					log.Warn("cascade access denied on edit target", slog.Int64("user_id", userID), slog.Int64("target_org_id", *req.OrganizationID))
					render.Status(r, http.StatusForbidden)
					render.JSON(w, r, resp.Forbidden("Нет доступа к целевой организации"))
					return
				}
				log.Error("cascade access check failed", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
				return
			}
		}

		storageReq := dto.EditShutdownRequest{
			OrganizationID:      req.OrganizationID,
			StartTime:           req.StartTime,
			EndTime:             req.EndTime,
			Reason:              req.Reason,
			GenerationLossMwh:   req.GenerationLossMwh,
			ReportedByContactID: req.ReportedByContactID,

			IdleDischargeVolumeThousandM3: req.IdleDischargeVolume,
			CreatedByUserID:               userID,
			FileIDs:                       req.FileIDs,
		}

		err = editor.EditShutdown(r.Context(), id, storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (org_id not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to update shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update shutdown"))
			return
		}

		// Update file links if explicitly requested
		if req.FileIDs != nil {
			if err := editor.UnlinkShutdownFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}
			if len(req.FileIDs) > 0 {
				if err := editor.LinkShutdownFiles(r.Context(), id, req.FileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		targetOrgID := curOrgID
		if req.OrganizationID != nil {
			targetOrgID = *req.OrganizationID
		}
		log.Info("shutdown updated successfully",
			slog.Int64("id", id),
			slog.Int64("user_id", userID),
			slog.Int64("target_org_id", targetOrgID),
		)
		render.JSON(w, r, resp.OK())
	}
}
