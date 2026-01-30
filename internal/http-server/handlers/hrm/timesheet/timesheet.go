package timesheet

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// TimesheetRepository defines the interface for timesheet operations
type TimesheetRepository interface {
	AddTimesheet(ctx context.Context, req hrm.AddTimesheetRequest) (int64, error)
	GetTimesheetByID(ctx context.Context, id int64) (*hrmmodel.Timesheet, error)
	GetTimesheets(ctx context.Context, filter hrm.TimesheetFilter) ([]*hrmmodel.Timesheet, error)
	UpdateTimesheetSummary(ctx context.Context, id int64, req hrm.EditTimesheetRequest) error
	SubmitTimesheet(ctx context.Context, id int64) error
	ApproveTimesheet(ctx context.Context, id int64, approvedBy int64, approved bool, rejectionReason *string) error
	DeleteTimesheet(ctx context.Context, id int64) error
}

// TimesheetEntryRepository defines the interface for timesheet entry operations
type TimesheetEntryRepository interface {
	AddTimesheetEntry(ctx context.Context, req hrm.AddTimesheetEntryRequest) (int64, error)
	GetTimesheetEntries(ctx context.Context, timesheetID int64) ([]*hrmmodel.TimesheetEntry, error)
	EditTimesheetEntry(ctx context.Context, id int64, req hrm.EditTimesheetEntryRequest) error
	DeleteTimesheetEntry(ctx context.Context, id int64) error
}

// HolidayRepository defines the interface for holiday operations
type HolidayRepository interface {
	AddHoliday(ctx context.Context, req hrm.AddHolidayRequest) (int, error)
	GetHolidays(ctx context.Context, year int) ([]*hrmmodel.Holiday, error)
	DeleteHoliday(ctx context.Context, id int) error
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Timesheet Handlers ---

// GetTimesheets returns timesheets with filters
func GetTimesheets(log *slog.Logger, repo TimesheetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.GetTimesheets"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.TimesheetFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err == nil {
				filter.EmployeeID = &val
			}
		}

		if yearStr := q.Get("year"); yearStr != "" {
			val, err := strconv.Atoi(yearStr)
			if err == nil {
				filter.Year = &val
			}
		}

		if monthStr := q.Get("month"); monthStr != "" {
			val, err := strconv.Atoi(monthStr)
			if err == nil {
				filter.Month = &val
			}
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if deptIDStr := q.Get("department_id"); deptIDStr != "" {
			val, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err == nil {
				filter.DepartmentID = &val
			}
		}

		timesheets, err := repo.GetTimesheets(r.Context(), filter)
		if err != nil {
			log.Error("failed to get timesheets", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve timesheets"))
			return
		}

		log.Info("successfully retrieved timesheets", slog.Int("count", len(timesheets)))
		render.JSON(w, r, timesheets)
	}
}

// GetTimesheetByID returns a timesheet by ID
func GetTimesheetByID(log *slog.Logger, repo TimesheetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.GetTimesheetByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		timesheet, err := repo.GetTimesheetByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("timesheet not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Timesheet not found"))
				return
			}
			log.Error("failed to get timesheet", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve timesheet"))
			return
		}

		log.Info("successfully retrieved timesheet", slog.Int64("id", timesheet.ID))
		render.JSON(w, r, timesheet)
	}
}

// AddTimesheet creates a new timesheet
func AddTimesheet(log *slog.Logger, repo TimesheetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.AddTimesheet"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddTimesheetRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddTimesheet(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate timesheet")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Timesheet for this employee/year/month already exists"))
				return
			}
			log.Error("failed to add timesheet", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add timesheet"))
			return
		}

		log.Info("timesheet added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditTimesheet updates a timesheet
func EditTimesheet(log *slog.Logger, repo TimesheetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.EditTimesheet"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditTimesheetRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.UpdateTimesheetSummary(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("timesheet not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Timesheet not found"))
				return
			}
			log.Error("failed to update timesheet", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update timesheet"))
			return
		}

		log.Info("timesheet updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// SubmitTimesheet submits a timesheet for approval
func SubmitTimesheet(log *slog.Logger, repo TimesheetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.SubmitTimesheet"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.SubmitTimesheet(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("timesheet not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Timesheet not found"))
				return
			}
			log.Error("failed to submit timesheet", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to submit timesheet"))
			return
		}

		log.Info("timesheet submitted", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// ApproveTimesheetRequest represents request to approve timesheet
type ApproveTimesheetRequest struct {
	Approved        bool    `json:"approved"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// ApproveTimesheet approves or rejects a timesheet
func ApproveTimesheet(log *slog.Logger, repo TimesheetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.ApproveTimesheet"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req ApproveTimesheetRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Get approver ID from JWT claims
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}
		approverID := claims.UserID

		err = repo.ApproveTimesheet(r.Context(), id, approverID, req.Approved, req.RejectionReason)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("timesheet not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Timesheet not found"))
				return
			}
			log.Error("failed to approve timesheet", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to approve timesheet"))
			return
		}

		action := "approved"
		if !req.Approved {
			action = "rejected"
		}
		log.Info("timesheet "+action, slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteTimesheet deletes a timesheet
func DeleteTimesheet(log *slog.Logger, repo TimesheetRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.DeleteTimesheet"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteTimesheet(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("timesheet not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Timesheet not found"))
				return
			}
			log.Error("failed to delete timesheet", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete timesheet"))
			return
		}

		log.Info("timesheet deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Timesheet Entry Handlers ---

// GetEntries returns timesheet entries for a specific timesheet
func GetEntries(log *slog.Logger, repo TimesheetEntryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.GetEntries"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()
		tsIDStr := q.Get("timesheet_id")
		if tsIDStr == "" {
			log.Warn("missing required 'timesheet_id' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Missing required 'timesheet_id' parameter"))
			return
		}

		timesheetID, err := strconv.ParseInt(tsIDStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'timesheet_id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'timesheet_id' parameter"))
			return
		}

		entries, err := repo.GetTimesheetEntries(r.Context(), timesheetID)
		if err != nil {
			log.Error("failed to get timesheet entries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve timesheet entries"))
			return
		}

		log.Info("successfully retrieved timesheet entries", slog.Int("count", len(entries)))
		render.JSON(w, r, entries)
	}
}

// AddEntry creates a new timesheet entry
func AddEntry(log *slog.Logger, repo TimesheetEntryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.AddEntry"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddTimesheetEntryRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddTimesheetEntry(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid timesheet_id"))
				return
			}
			log.Error("failed to add timesheet entry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add timesheet entry"))
			return
		}

		log.Info("timesheet entry added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditEntry updates a timesheet entry
func EditEntry(log *slog.Logger, repo TimesheetEntryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.EditEntry"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditTimesheetEntryRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditTimesheetEntry(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("timesheet entry not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Timesheet entry not found"))
				return
			}
			log.Error("failed to update timesheet entry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update timesheet entry"))
			return
		}

		log.Info("timesheet entry updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteEntry deletes a timesheet entry
func DeleteEntry(log *slog.Logger, repo TimesheetEntryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.DeleteEntry"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteTimesheetEntry(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("timesheet entry not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Timesheet entry not found"))
				return
			}
			log.Error("failed to delete timesheet entry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete timesheet entry"))
			return
		}

		log.Info("timesheet entry deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Holiday Handlers ---

// GetHolidays returns holidays for a year
func GetHolidays(log *slog.Logger, repo HolidayRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.GetHolidays"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		yearStr := r.URL.Query().Get("year")
		if yearStr == "" {
			log.Warn("missing 'year' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("'year' parameter is required"))
			return
		}

		year, err := strconv.Atoi(yearStr)
		if err != nil {
			log.Warn("invalid 'year' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'year' parameter"))
			return
		}

		holidays, err := repo.GetHolidays(r.Context(), year)
		if err != nil {
			log.Error("failed to get holidays", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve holidays"))
			return
		}

		log.Info("successfully retrieved holidays", slog.Int("count", len(holidays)))
		render.JSON(w, r, holidays)
	}
}

// AddHoliday creates a new holiday
func AddHoliday(log *slog.Logger, repo HolidayRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.AddHoliday"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddHolidayRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddHoliday(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate holiday")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Holiday for this date already exists"))
				return
			}
			log.Error("failed to add holiday", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add holiday"))
			return
		}

		log.Info("holiday added", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: int64(id)})
	}
}

// DeleteHoliday deletes a holiday
func DeleteHoliday(log *slog.Logger, repo HolidayRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.timesheet.DeleteHoliday"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteHoliday(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("holiday not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Holiday not found"))
				return
			}
			log.Error("failed to delete holiday", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete holiday"))
			return
		}

		log.Info("holiday deleted", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
