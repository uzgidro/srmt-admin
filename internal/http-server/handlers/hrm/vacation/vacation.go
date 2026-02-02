package vacation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	"srmt-admin/internal/http-server/handlers/hrm/authz"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// VacationTypeRepository defines the interface for vacation type operations
type VacationTypeRepository interface {
	AddVacationType(ctx context.Context, req hrm.AddVacationTypeRequest) (int, error)
	GetAllVacationTypes(ctx context.Context, activeOnly bool) ([]*hrmmodel.VacationType, error)
	EditVacationType(ctx context.Context, id int, req hrm.EditVacationTypeRequest) error
	DeleteVacationType(ctx context.Context, id int) error
}

// VacationBalanceRepository defines the interface for vacation balance operations
type VacationBalanceRepository interface {
	AddVacationBalance(ctx context.Context, req hrm.AddVacationBalanceRequest) (int64, error)
	GetVacationBalances(ctx context.Context, filter hrm.VacationBalanceFilter) ([]*hrmmodel.VacationBalance, error)
	EditVacationBalance(ctx context.Context, id int64, req hrm.EditVacationBalanceRequest) error
}

// VacationRepository defines the interface for vacation request operations
type VacationRepository interface {
	AddVacation(ctx context.Context, req hrm.AddVacationRequest) (int64, error)
	GetVacationByID(ctx context.Context, id int64) (*hrmmodel.Vacation, error)
	GetVacations(ctx context.Context, filter hrm.VacationFilter) ([]*hrmmodel.Vacation, error)
	EditVacation(ctx context.Context, id int64, req hrm.EditVacationRequest) error
	ApproveVacation(ctx context.Context, id int64, approverID int64, approved bool, rejectionReason *string) error
	CancelVacation(ctx context.Context, id int64) error
	DeleteVacation(ctx context.Context, id int64) error
	GetVacationCalendar(ctx context.Context, filter hrm.VacationCalendarFilter) ([]*hrmmodel.VacationCalendarEntry, error)
}

// EmployeeAccessChecker provides methods for checking employee data access
type EmployeeAccessChecker interface {
	GetEmployeeIDByUserID(ctx context.Context, userID int64) (int64, error)
	IsManagerOf(ctx context.Context, managerEmployeeID, employeeID int64) (bool, error)
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Vacation Type Handlers ---

// GetTypes returns all vacation types
func GetTypes(log *slog.Logger, repo VacationTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetTypes"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		activeOnly := r.URL.Query().Get("active_only") == "true"

		types, err := repo.GetAllVacationTypes(r.Context(), activeOnly)
		if err != nil {
			log.Error("failed to get vacation types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacation types"))
			return
		}

		log.Info("successfully retrieved vacation types", slog.Int("count", len(types)))
		render.JSON(w, r, types)
	}
}

// AddType creates a new vacation type
func AddType(log *slog.Logger, repo VacationTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.AddType"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddVacationTypeRequest
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

		id, err := repo.AddVacationType(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate vacation type code", "code", req.Code)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Vacation type with this code already exists"))
				return
			}
			log.Error("failed to add vacation type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add vacation type"))
			return
		}

		log.Info("vacation type added", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: int64(id)})
	}
}

// EditType updates a vacation type
func EditType(log *slog.Logger, repo VacationTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.EditType"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditVacationTypeRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditVacationType(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation type not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation type not found"))
				return
			}
			log.Error("failed to update vacation type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update vacation type"))
			return
		}

		log.Info("vacation type updated", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteType deletes a vacation type
func DeleteType(log *slog.Logger, repo VacationTypeRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.DeleteType"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteVacationType(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation type not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation type not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("vacation type has dependencies", slog.Int("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: vacation type is in use"))
				return
			}
			log.Error("failed to delete vacation type", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete vacation type"))
			return
		}

		log.Info("vacation type deleted", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Vacation Balance Handlers ---

// GetBalances returns vacation balances with filters
func GetBalances(log *slog.Logger, repo VacationBalanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetBalances"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.VacationBalanceFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'employee_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'employee_id' parameter"))
				return
			}
			filter.EmployeeID = &val
		}

		if typeIDStr := q.Get("vacation_type_id"); typeIDStr != "" {
			val, err := strconv.Atoi(typeIDStr)
			if err != nil {
				log.Warn("invalid 'vacation_type_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'vacation_type_id' parameter"))
				return
			}
			filter.VacationTypeID = &val
		}

		if yearStr := q.Get("year"); yearStr != "" {
			val, err := strconv.Atoi(yearStr)
			if err != nil {
				log.Warn("invalid 'year' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'year' parameter"))
				return
			}
			filter.Year = &val
		}

		balances, err := repo.GetVacationBalances(r.Context(), filter)
		if err != nil {
			log.Error("failed to get vacation balances", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacation balances"))
			return
		}

		log.Info("successfully retrieved vacation balances", slog.Int("count", len(balances)))
		render.JSON(w, r, balances)
	}
}

// AddBalance creates or updates vacation balance
func AddBalance(log *slog.Logger, repo VacationBalanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.AddBalance"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddVacationBalanceRequest
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

		id, err := repo.AddVacationBalance(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id or vacation_type_id"))
				return
			}
			log.Error("failed to add vacation balance", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add vacation balance"))
			return
		}

		log.Info("vacation balance added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditBalance updates vacation balance
func EditBalance(log *slog.Logger, repo VacationBalanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.EditBalance"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditVacationBalanceRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditVacationBalance(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation balance not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation balance not found"))
				return
			}
			log.Error("failed to update vacation balance", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update vacation balance"))
			return
		}

		log.Info("vacation balance updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// --- Vacation Request Handlers ---

// GetAll returns vacation requests with filters
// Access control: HR/admin/manager can see all or subordinates, others see only their own
func GetAll(log *slog.Logger, repo VacationRepository, accessChecker EmployeeAccessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Authentication required"))
			return
		}

		var filter hrm.VacationFilter
		q := r.URL.Query()

		// Check if user has privileged access (HR or admin)
		hasPrivilegedAccess := authz.HasAnyRole(claims, "hr", "admin")

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'employee_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'employee_id' parameter"))
				return
			}

			// Check access permission for the requested employee's vacations
			if !hasPrivilegedAccess {
				canAccess, _, err := authz.CanAccessEmployeeData(r.Context(), claims, val, accessChecker)
				if err != nil {
					log.Error("failed to check access permission", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
					return
				}
				if !canAccess {
					log.Warn("access denied to employee vacations", slog.Int64("target_employee_id", val), slog.Int64("user_id", claims.UserID))
					render.Status(r, http.StatusForbidden)
					render.JSON(w, r, resp.Forbidden("Access denied to this employee's vacation data"))
					return
				}
			}
			filter.EmployeeID = &val
		} else if !hasPrivilegedAccess {
			// Non-privileged users must specify employee_id (their own)
			currentEmpID, err := accessChecker.GetEmployeeIDByUserID(r.Context(), claims.UserID)
			if err != nil {
				log.Warn("user has no employee record", slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("Access denied - no employee record found"))
				return
			}
			filter.EmployeeID = &currentEmpID
		}

		if typeIDStr := q.Get("vacation_type_id"); typeIDStr != "" {
			val, err := strconv.Atoi(typeIDStr)
			if err != nil {
				log.Warn("invalid 'vacation_type_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'vacation_type_id' parameter"))
				return
			}
			filter.VacationTypeID = &val
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if fromDateStr := q.Get("from_date"); fromDateStr != "" {
			val, err := time.Parse(time.DateOnly, fromDateStr)
			if err != nil {
				log.Warn("invalid 'from_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'from_date' parameter, use YYYY-MM-DD"))
				return
			}
			filter.FromDate = &val
		}

		if toDateStr := q.Get("to_date"); toDateStr != "" {
			val, err := time.Parse(time.DateOnly, toDateStr)
			if err != nil {
				log.Warn("invalid 'to_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'to_date' parameter, use YYYY-MM-DD"))
				return
			}
			filter.ToDate = &val
		}

		if deptIDStr := q.Get("department_id"); deptIDStr != "" {
			val, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'department_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'department_id' parameter"))
				return
			}
			filter.DepartmentID = &val
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, err := strconv.Atoi(limitStr)
			if err != nil {
				log.Warn("invalid 'limit' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'limit' parameter"))
				return
			}
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, err := strconv.Atoi(offsetStr)
			if err != nil {
				log.Warn("invalid 'offset' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'offset' parameter"))
				return
			}
			filter.Offset = val
		}

		vacations, err := repo.GetVacations(r.Context(), filter)
		if err != nil {
			log.Error("failed to get vacations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacations"))
			return
		}

		log.Info("successfully retrieved vacations", slog.Int("count", len(vacations)))
		render.JSON(w, r, vacations)
	}
}

// GetByID returns a vacation request by ID
// Access control: HR/admin/manager can see all, others see only their own
func GetByID(log *slog.Logger, repo VacationRepository, accessChecker EmployeeAccessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Authentication required"))
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

		vacation, err := repo.GetVacationByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			log.Error("failed to get vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacation"))
			return
		}

		// Check access permission for the vacation's employee
		if !authz.HasAnyRole(claims, "hr", "admin") {
			canAccess, _, err := authz.CanAccessEmployeeData(r.Context(), claims, vacation.EmployeeID, accessChecker)
			if err != nil {
				log.Error("failed to check access permission", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
				return
			}
			if !canAccess {
				log.Warn("access denied to vacation record", slog.Int64("vacation_id", id), slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("Access denied to this vacation record"))
				return
			}
		}

		log.Info("successfully retrieved vacation", slog.Int64("id", vacation.ID))
		render.JSON(w, r, vacation)
	}
}

// Add creates a new vacation request
func Add(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddVacationRequest
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

		id, err := repo.AddVacation(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id, vacation_type_id, or substitute_employee_id"))
				return
			}
			log.Error("failed to add vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add vacation"))
			return
		}

		log.Info("vacation added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// Edit updates a vacation request
func Edit(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditVacationRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditVacation(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			log.Error("failed to update vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update vacation"))
			return
		}

		log.Info("vacation updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// Approve approves or rejects a vacation request
func Approve(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Approve"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.ApproveVacationRequest
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

		err = repo.ApproveVacation(r.Context(), id, approverID, req.Approved, req.RejectionReason)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			log.Error("failed to approve/reject vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to process vacation request"))
			return
		}

		action := "approved"
		if !req.Approved {
			action = "rejected"
		}
		log.Info("vacation "+action, slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// Cancel cancels a vacation request
func Cancel(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Cancel"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.CancelVacation(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			log.Error("failed to cancel vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to cancel vacation"))
			return
		}

		log.Info("vacation cancelled", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// Delete deletes a vacation request
func Delete(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.Delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteVacation(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation not found"))
				return
			}
			log.Error("failed to delete vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete vacation"))
			return
		}

		log.Info("vacation deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// GetCalendar returns vacation calendar for a month
func GetCalendar(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.vacation.GetCalendar"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		q := r.URL.Query()

		yearStr := q.Get("year")
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

		monthStr := q.Get("month")
		if monthStr == "" {
			log.Warn("missing 'month' parameter")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("'month' parameter is required"))
			return
		}
		month, err := strconv.Atoi(monthStr)
		if err != nil || month < 1 || month > 12 {
			log.Warn("invalid 'month' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'month' parameter"))
			return
		}

		filter := hrm.VacationCalendarFilter{
			Year:  year,
			Month: month,
		}

		if deptIDStr := q.Get("department_id"); deptIDStr != "" {
			val, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'department_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'department_id' parameter"))
				return
			}
			filter.DepartmentID = &val
		}

		if orgIDStr := q.Get("organization_id"); orgIDStr != "" {
			val, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'organization_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
				return
			}
			filter.OrganizationID = &val
		}

		calendar, err := repo.GetVacationCalendar(r.Context(), filter)
		if err != nil {
			log.Error("failed to get vacation calendar", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacation calendar"))
			return
		}

		log.Info("successfully retrieved vacation calendar", slog.Int("entries", len(calendar)))
		render.JSON(w, r, calendar)
	}
}
