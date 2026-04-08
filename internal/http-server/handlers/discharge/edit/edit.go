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
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"strconv"
	"time"
)

type Request struct {
	StartedAt      *time.Time `json:"started_at,omitempty"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	FlowRate       *float64   `json:"flow_rate,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	Approved       *bool      `json:"approved,omitempty"`
	OrganizationID *int64     `json:"organization_id,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
}

type DischargeEditor interface {
	EditDischarge(ctx context.Context, id, approvedByID int64, startTime, endTime *time.Time, flowRate *float64, reason *string, approved *bool, organizationID *int64) error
	UnlinkDischargeFiles(ctx context.Context, dischargeID int64) error
	LinkDischargeFiles(ctx context.Context, dischargeID int64, fileIDs []int64) error
}

type DischargeOrgGetter interface {
	GetDischargeOrgID(ctx context.Context, id int64) (int64, error)
}

func New(log *slog.Logger, editor DischargeEditor, orgGetter DischargeOrgGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.patch.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		dischargeID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			log.Warn("invalid discharge ID format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid discharge ID"))
			return
		}

		// Check organization access
		resourceOrgID, err := orgGetter.GetDischargeOrgID(r.Context(), dischargeID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Discharge not found"))
				return
			}
			log.Error("failed to get discharge org_id", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to check access"))
			return
		}
		if err := auth.CheckOrgAccess(r.Context(), resourceOrgID); err != nil {
			log.Warn("org access denied for discharge edit", sl.Err(err))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = editor.EditDischarge(r.Context(), dischargeID, userID, req.StartedAt, req.EndedAt, req.FlowRate, req.Reason, req.Approved, req.OrganizationID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("discharge not found", "id", dischargeID)
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Discharge not found"))
				return
			}
			log.Error("failed to edit discharge", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to edit discharge"))
			return
		}

		// Update file links if explicitly requested
		if req.FileIDs != nil {
			if err := editor.UnlinkDischargeFiles(r.Context(), dischargeID); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}
			if len(req.FileIDs) > 0 {
				if err := editor.LinkDischargeFiles(r.Context(), dischargeID, req.FileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("discharge updated successfully", slog.Int64("id", dischargeID))
		render.JSON(w, r, resp.OK())
	}
}
