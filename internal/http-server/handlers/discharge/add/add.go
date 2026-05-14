package add

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/dayrotation/cutoffs"
	"srmt-admin/internal/storage"
	"time"
)

type Request struct {
	OrganizationID int64      `json:"organization_id" validate:"required"`
	StartedAt      time.Time  `json:"started_at" validate:"required"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	FlowRate       float64    `json:"flow_rate" validate:"required,gt=0"`
	Reason         *string    `json:"reason,omitempty"`
	FileIDs        []int64    `json:"file_ids,omitempty"`
	Force          bool       `json:"force,omitempty"`
}

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

type DischargeAdder interface {
	AddDischarge(ctx context.Context, orgID, createdByID int64, startTime time.Time, endTime *time.Time, flowRate float64, reason *string) (int64, error)
	LinkDischargeFiles(ctx context.Context, dischargeID int64, fileIDs []int64) error
}

type OngoingChecker interface {
	EnsureNoOngoingDischarge(ctx context.Context, orgID int64, force bool, newStartTime time.Time) error
}

// BackdateRotator emulates the dayrotation ticker for a freshly created
// discharge whose start_time predates one or more 05:00 cutoffs. Returns the
// final clone's ID; if cutoffs is empty, returns the input ID unchanged.
type BackdateRotator interface {
	RotateBackdatedDischarge(ctx context.Context, dischargeID int64, cutoffs []time.Time) (int64, error)
}

func New(log *slog.Logger, adder DischargeAdder, checker OngoingChecker, rotator BackdateRotator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.discharge.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Check organization access
		if err := auth.CheckOrgAccess(r.Context(), req.OrganizationID); err != nil {
			log.Warn("org access denied for discharge add", sl.Err(err))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		// Check for ongoing discharge conflict
		if err := checker.EnsureNoOngoingDischarge(r.Context(), req.OrganizationID, req.Force, req.StartedAt); err != nil {
			if errors.Is(err, storage.ErrOngoingDischargeExists) {
				log.Warn("ongoing discharge exists", "org_id", req.OrganizationID)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Для данной организации уже существует незавершенный холостой сброс"))
				return
			}
			if errors.Is(err, storage.ErrDischargeEndBeforeStart) {
				log.Warn("new start time is before existing discharge start time", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Время начала нового сброса раньше начала текущего незавершённого сброса"))
				return
			}
			log.Error("failed to check ongoing discharge", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to check ongoing discharges"))
			return
		}

		// Pre-compute backdate cutoffs from started_at. This is a pure
		// computation (no DB) — running it BEFORE AddDischarge ensures the
		// ErrBackdateTooOld case rejects the request without leaving an
		// orphan ongoing discharge in the DB.
		cuts, err := cutoffs.Compute(req.StartedAt, time.Now(), 5, loc)
		if err != nil {
			if errors.Is(err, cutoffs.ErrBackdateTooOld) {
				log.Warn("started_at too far in the past", "started_at", req.StartedAt)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("started_at is more than 90 days in the past"))
				return
			}
			log.Error("failed to compute cutoffs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to compute backdate cutoffs"))
			return
		}

		// Create discharge
		id, err := adder.AddDischarge(r.Context(), req.OrganizationID, userID, req.StartedAt, req.EndedAt, req.FlowRate, req.Reason)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("duplicate ongoing discharge (race condition)", "org_id", req.OrganizationID)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Для данной организации уже существует незавершенный холостой сброс"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("organization not found", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to add discharge", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add discharge"))
			return
		}

		// Link files if provided
		if len(req.FileIDs) > 0 {
			if err := adder.LinkDischargeFiles(r.Context(), id, req.FileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
				// Don't fail the request, just log the error
			}
		}

		// Backdate rotation: if started_at predates one or more 05:00 local
		// cutoffs, emulate the dayrotation ticker for each missed cutoff so
		// the record is split per operational day. Mirrors what the scheduled
		// job does daily (RotateDayBoundary). The handler returns the final
		// clone's ID — that's the row that remains "current ongoing" and is
		// what the frontend will see at GET /discharges/current.
		if len(cuts) > 0 {
			finalID, err := rotator.RotateBackdatedDischarge(r.Context(), id, cuts)
			if err != nil {
				log.Error("failed to rotate backdated discharge", sl.Err(err), slog.Int64("original_id", id), slog.Int("cutoffs", len(cuts)))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to rotate backdated discharge"))
				return
			}
			log.Info("backdate rotation applied",
				slog.Int64("original_id", id),
				slog.Int64("final_id", finalID),
				slog.Int("cutoffs", len(cuts)),
			)
			id = finalID
		}

		log.Info("discharge added successfully",
			slog.Int64("id", id),
			slog.Int("files", len(req.FileIDs)),
		)

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{
			Response: resp.Created(),
			ID:       id,
		})
	}
}
