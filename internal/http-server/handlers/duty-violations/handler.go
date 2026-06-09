// Package dutyviolationshandler exposes the HTTP layer for duty-officer
// violation records. Each handler depends on a narrow service interface
// (one method per endpoint) so tests can mock the exact surface they need.
package dutyviolationshandler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	dvmodel "srmt-admin/internal/lib/model/duty-violations"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

// --- Local interfaces — handler depends only on what it calls ---

type ServiceCreator interface {
	Create(ctx context.Context, req dvmodel.CreateRequest, createdByUserID int64) (*dvmodel.DutyViolation, error)
}

type ServiceLister interface {
	List(ctx context.Context, f dvmodel.ListFilter) ([]*dvmodel.DutyViolation, error)
}

// ServiceUpdater needs GetByID too: PATCH must authorize against the
// record's CURRENT organization_id, not against whatever the request body
// claims — otherwise a caller with access to org A could move (or just
// edit) a record belonging to org B by passing organization_id=A.
type ServiceUpdater interface {
	GetByID(ctx context.Context, id int64) (*dvmodel.DutyViolation, error)
	Update(ctx context.Context, id int64, req dvmodel.UpdateRequest) (*dvmodel.DutyViolation, error)
}

// ServiceDeleter needs GetByID for the same reason: without checking the
// existing record's org, any caller with the right role could delete any
// record by guessing the id (IDOR).
type ServiceDeleter interface {
	GetByID(ctx context.Context, id int64) (*dvmodel.DutyViolation, error)
	Delete(ctx context.Context, id int64) error
}

// validate is shared between all handlers — validator instances are
// thread-safe and cache reflection results, so reusing one cuts setup
// cost on every request.
var validate = validator.New()

// --- POST /duty-violations ---

func Add(log *slog.Logger, svc ServiceCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.duty-violations.Add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req dvmodel.CreateRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}
		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Warn("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Authorization BEFORE the service call: a caller without access
		// to the organization should not even cause an INSERT attempt.
		if err := auth.CheckOrgAccess(r.Context(), req.OrganizationID); err != nil {
			log.Warn("org access denied", sl.Err(err), slog.Int64("organization_id", req.OrganizationID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		dv, err := svc.Create(r.Context(), req, userID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("organization or file not found", sl.Err(err))
				render.Status(r, http.StatusUnprocessableEntity)
				render.JSON(w, r, resp.BadRequest("Organization or file does not exist"))
				return
			}
			// CHECK constraint catches whitespace-only name/reason and
			// end_time<=start_time that slip past the validator (e.g. when
			// callers strip the validator with unconventional inputs).
			if errors.Is(err, storage.ErrCheckConstraintViolation) {
				log.Warn("CHECK constraint violation", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Field values violate constraints (blank text or end_time <= start_time)"))
				return
			}
			log.Error("failed to create duty violation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create record"))
			return
		}

		log.Info("duty violation created", slog.Int64("id", dv.ID))
		render.Status(r, http.StatusOK)
		render.JSON(w, r, dv)
	}
}

// --- GET /duty-violations?organization_id=N&from=YYYY-MM-DD&to=YYYY-MM-DD ---

// List accepts loc *time.Location so date filters can be interpreted as
// operational days (05:00 local boundary) instead of raw UTC midnight —
// matches the convention used by incidents/discharges/ges-report.
func List(log *slog.Logger, svc ServiceLister, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.duty-violations.List"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		f, err := parseListFilter(r, loc)
		if err != nil {
			log.Warn("invalid filter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(err.Error()))
			return
		}

		// Tenant scoping for non-privileged callers:
		// - If a specific organization_id was requested, CheckOrgAccess
		//   rejects foreign orgs with 403 (sc/rais pass through).
		// - If no organization_id was passed, a non-privileged caller
		//   would otherwise see every record across every org. Force the
		//   filter to their first claims-listed org (the typical case is
		//   single-org membership). This is a safety net: today the
		//   router only lets sc/rais reach here, but when reservoir-class
		//   roles are added the handler is already safe.
		if !isPrivilegedCaller(r.Context()) {
			if f.OrganizationID != nil {
				if err := auth.CheckOrgAccess(r.Context(), *f.OrganizationID); err != nil {
					log.Warn("org access denied on list filter",
						sl.Err(err), slog.Int64("organization_id", *f.OrganizationID))
					render.Status(r, http.StatusForbidden)
					render.JSON(w, r, resp.Forbidden("Access denied"))
					return
				}
			} else {
				claims, ok := mwauth.ClaimsFromContext(r.Context())
				if !ok || claims == nil || len(claims.OrganizationIDs) == 0 {
					// No org assignment → no records to show.
					render.JSON(w, r, []*dvmodel.DutyViolation{})
					return
				}
				ownOrg := claims.OrganizationIDs[0]
				f.OrganizationID = &ownOrg
			}
		}

		list, err := svc.List(r.Context(), f)
		if err != nil {
			log.Error("failed to list duty violations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to list records"))
			return
		}

		// Always render a JSON array, never null — frontend never has to
		// guard `.map()` against a missing list.
		if list == nil {
			list = []*dvmodel.DutyViolation{}
		}
		render.JSON(w, r, list)
	}
}

// --- PATCH /duty-violations/{id} ---

func Edit(log *slog.Logger, svc ServiceUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.duty-violations.Edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := parseIDParam(r)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid id"))
			return
		}

		var req dvmodel.UpdateRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}
		if err := validate.Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Warn("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// IDOR defense: load the existing record FIRST and authorize against
		// its current organization_id, not against the request body's
		// claim. Otherwise a caller with access to org A could PATCH a
		// record owned by org B simply by setting organization_id=A in
		// the body. If the caller wants to move the record to a different
		// org, they must also have access to the destination — checked
		// separately below.
		existing, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Record not found"))
				return
			}
			log.Error("failed to load existing record", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to load record"))
			return
		}
		if err := auth.CheckOrgAccess(r.Context(), existing.OrganizationID); err != nil {
			log.Warn("org access denied on existing record",
				sl.Err(err), slog.Int64("existing_org", existing.OrganizationID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}
		// Cross-org transfer also requires access to the destination org.
		// For sc/rais this is a no-op (full access); for any future role
		// confined to specific orgs, it stops reassignment to a foreign org.
		if req.OrganizationID != existing.OrganizationID {
			if err := auth.CheckOrgAccess(r.Context(), req.OrganizationID); err != nil {
				log.Warn("org access denied on target org of transfer",
					sl.Err(err), slog.Int64("target_org", req.OrganizationID))
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("Access denied"))
				return
			}
		}

		dv, err := svc.Update(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Record not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				render.Status(r, http.StatusUnprocessableEntity)
				render.JSON(w, r, resp.BadRequest("Organization or file does not exist"))
				return
			}
			if errors.Is(err, storage.ErrCheckConstraintViolation) {
				log.Warn("CHECK constraint violation", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Field values violate constraints (blank text or end_time <= start_time)"))
				return
			}
			log.Error("failed to update duty violation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update record"))
			return
		}

		log.Info("duty violation updated", slog.Int64("id", id))
		render.JSON(w, r, dv)
	}
}

// --- DELETE /duty-violations/{id} ---

func Delete(log *slog.Logger, svc ServiceDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.duty-violations.Delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := parseIDParam(r)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid id"))
			return
		}

		// IDOR defense: load the record and authorize against its org
		// before deleting. Without this, any caller in the route group
		// could delete records belonging to organizations they don't own
		// (today only sc/rais reach this handler so the leak is latent;
		// the check makes the feature safe to expose to other roles).
		existing, err := svc.GetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Record not found"))
				return
			}
			log.Error("failed to load existing record", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to load record"))
			return
		}
		if err := auth.CheckOrgAccess(r.Context(), existing.OrganizationID); err != nil {
			log.Warn("org access denied on delete",
				sl.Err(err), slog.Int64("existing_org", existing.OrganizationID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
		}

		if err := svc.Delete(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Record not found"))
				return
			}
			log.Error("failed to delete duty violation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete record"))
			return
		}

		log.Info("duty violation deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
		render.JSON(w, r, resp.Delete())
	}
}

// isPrivilegedCaller reports whether the caller is sc or rais (full
// cross-org access). Used by the List handler to decide whether to scope
// the query to the caller's organizations — keeping the rule local to
// this file so changes don't ripple across modules.
func isPrivilegedCaller(ctx context.Context) bool {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return false
	}
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			return true
		}
	}
	return false
}

// --- helpers (file-local) ---

// parseIDParam reads {id} from the chi URL pattern. We do not accept 0 or
// negative — those can't match a real serial primary key.
func parseIDParam(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

// parseListFilter reads the optional ?organization_id, ?from and ?to query
// parameters. Dates are interpreted as operational days (05:00 local
// boundary, Asia/Tashkent by default — same as incidents/discharges).
//
// The to-boundary is the start of the day AFTER `to`, so the repo can
// filter as `start_time < to` (half-open interval). That convention is
// shared with `cutoffs.Compute` — a record landing exactly on the cutoff
// belongs to the next op-day, not the previous one.
func parseListFilter(r *http.Request, loc *time.Location) (dvmodel.ListFilter, error) {
	var f dvmodel.ListFilter
	q := r.URL.Query()

	if s := q.Get("organization_id"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil || v <= 0 {
			return f, errors.New("invalid organization_id")
		}
		f.OrganizationID = &v
	}
	if s := q.Get("from"); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, loc)
		if err != nil {
			return f, errors.New("invalid from date, expected YYYY-MM-DD")
		}
		// День начинается в 05:00 местного времени.
		start := time.Date(t.Year(), t.Month(), t.Day(), 5, 0, 0, 0, loc)
		f.From = &start
	}
	if s := q.Get("to"); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, loc)
		if err != nil {
			return f, errors.New("invalid to date, expected YYYY-MM-DD")
		}
		// День заканчивается в 05:00 локального времени следующего дня.
		end := time.Date(t.Year(), t.Month(), t.Day(), 5, 0, 0, 0, loc).Add(24 * time.Hour)
		f.To = &end
	}
	return f, nil
}
