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

type ServiceUpdater interface {
	Update(ctx context.Context, id int64, req dvmodel.UpdateRequest) (*dvmodel.DutyViolation, error)
}

type ServiceDeleter interface {
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

func List(log *slog.Logger, svc ServiceLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.duty-violations.List"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		f, err := parseListFilter(r)
		if err != nil {
			log.Warn("invalid filter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest(err.Error()))
			return
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
		if err := auth.CheckOrgAccess(r.Context(), req.OrganizationID); err != nil {
			log.Warn("org access denied", sl.Err(err), slog.Int64("organization_id", req.OrganizationID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("Access denied"))
			return
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
// parameters. Each is independent — passing 1, 2 or none all work. Dates
// accept YYYY-MM-DD (interpreted as midnight UTC for `from`; end-of-day
// upper bound is the caller's job).
func parseListFilter(r *http.Request) (dvmodel.ListFilter, error) {
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
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return f, errors.New("invalid from date, expected YYYY-MM-DD")
		}
		f.From = &t
	}
	if s := q.Get("to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return f, errors.New("invalid to date, expected YYYY-MM-DD")
		}
		f.To = &t
	}
	return f, nil
}
