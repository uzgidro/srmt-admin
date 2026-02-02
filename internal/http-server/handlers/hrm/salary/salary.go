package salary

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

	"srmt-admin/internal/http-server/handlers/hrm/authz"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// SalaryStructureRepository defines the interface for salary structure operations
type SalaryStructureRepository interface {
	AddSalaryStructure(ctx context.Context, req hrm.AddSalaryStructureRequest) (int64, error)
	GetSalaryStructureByID(ctx context.Context, id int64) (*hrmmodel.SalaryStructure, error)
	GetSalaryStructures(ctx context.Context, filter hrm.SalaryStructureFilter) ([]*hrmmodel.SalaryStructure, error)
	EditSalaryStructure(ctx context.Context, id int64, req hrm.EditSalaryStructureRequest) error
	DeleteSalaryStructure(ctx context.Context, id int64) error
}

// SalaryRepository defines the interface for salary operations
type SalaryRepository interface {
	AddSalary(ctx context.Context, req hrm.AddSalaryRequest) (int64, error)
	GetSalaryByID(ctx context.Context, id int64) (*hrmmodel.Salary, error)
	GetSalaries(ctx context.Context, filter hrm.SalaryFilter) ([]*hrmmodel.Salary, error)
	ApproveSalary(ctx context.Context, id int64, approvedBy int64) error
	MarkSalaryPaid(ctx context.Context, id int64) error
	DeleteSalary(ctx context.Context, id int64) error
}

// BonusRepository defines the interface for bonus operations
type BonusRepository interface {
	AddBonus(ctx context.Context, req hrm.AddBonusRequest) (int64, error)
	GetBonuses(ctx context.Context, filter hrm.BonusFilter) ([]*hrmmodel.SalaryBonus, error)
	ApproveBonus(ctx context.Context, id int64, approvedBy int64) error
	DeleteBonus(ctx context.Context, id int64) error
}

// DeductionRepository defines the interface for deduction operations
type DeductionRepository interface {
	AddDeduction(ctx context.Context, req hrm.AddDeductionRequest) (int64, error)
	GetDeductions(ctx context.Context, filter hrm.DeductionFilter) ([]*hrmmodel.SalaryDeduction, error)
	DeleteDeduction(ctx context.Context, id int64) error
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

// --- Salary Structure Handlers ---

// GetStructures returns salary structures
// Access control: HR/admin can see all, others can only see their own salary structure
func GetStructures(log *slog.Logger, repo SalaryStructureRepository, accessChecker EmployeeAccessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetStructures"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Authentication required"))
			return
		}

		var filter hrm.SalaryStructureFilter
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

			// Check access permission for the requested employee's salary structure
			if !hasPrivilegedAccess {
				canAccess, err := authz.CanAccessSalaryData(r.Context(), claims, val, accessChecker)
				if err != nil {
					log.Error("failed to check access permission", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
					return
				}
				if !canAccess {
					log.Warn("access denied to employee salary structure", slog.Int64("target_employee_id", val), slog.Int64("user_id", claims.UserID))
					render.Status(r, http.StatusForbidden)
					render.JSON(w, r, resp.Forbidden("Access denied to this employee's salary structure"))
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

		filter.ActiveOnly = q.Get("active_only") == "true"

		structures, err := repo.GetSalaryStructures(r.Context(), filter)
		if err != nil {
			log.Error("failed to get salary structures", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary structures"))
			return
		}

		log.Info("successfully retrieved salary structures", slog.Int("count", len(structures)))
		render.JSON(w, r, structures)
	}
}

// AddStructure creates a new salary structure
func AddStructure(log *slog.Logger, repo SalaryStructureRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.AddStructure"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddSalaryStructureRequest
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

		id, err := repo.AddSalaryStructure(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id"))
				return
			}
			log.Error("failed to add salary structure", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add salary structure"))
			return
		}

		log.Info("salary structure added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditStructure updates a salary structure
func EditStructure(log *slog.Logger, repo SalaryStructureRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.EditStructure"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditSalaryStructureRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditSalaryStructure(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("salary structure not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Salary structure not found"))
				return
			}
			log.Error("failed to update salary structure", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update salary structure"))
			return
		}

		log.Info("salary structure updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteStructure deletes a salary structure
func DeleteStructure(log *slog.Logger, repo SalaryStructureRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.DeleteStructure"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteSalaryStructure(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("salary structure not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Salary structure not found"))
				return
			}
			log.Error("failed to delete salary structure", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete salary structure"))
			return
		}

		log.Info("salary structure deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Salary Handlers ---

// GetAll returns salaries with filters
// Access control: HR/admin can see all, others can only see their own salary
func GetAll(log *slog.Logger, repo SalaryRepository, accessChecker EmployeeAccessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetAll"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Authentication required"))
			return
		}

		var filter hrm.SalaryFilter
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

			// Check access permission for the requested employee's salary
			if !hasPrivilegedAccess {
				canAccess, err := authz.CanAccessSalaryData(r.Context(), claims, val, accessChecker)
				if err != nil {
					log.Error("failed to check access permission", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
					return
				}
				if !canAccess {
					log.Warn("access denied to employee salary", slog.Int64("target_employee_id", val), slog.Int64("user_id", claims.UserID))
					render.Status(r, http.StatusForbidden)
					render.JSON(w, r, resp.Forbidden("Access denied to this employee's salary data"))
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

		if monthStr := q.Get("month"); monthStr != "" {
			val, err := strconv.Atoi(monthStr)
			if err != nil {
				log.Warn("invalid 'month' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'month' parameter"))
				return
			}
			filter.Month = &val
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
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
			if err == nil {
				filter.Limit = val
			}
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, err := strconv.Atoi(offsetStr)
			if err == nil {
				filter.Offset = val
			}
		}

		salaries, err := repo.GetSalaries(r.Context(), filter)
		if err != nil {
			log.Error("failed to get salaries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salaries"))
			return
		}

		log.Info("successfully retrieved salaries", slog.Int("count", len(salaries)))
		render.JSON(w, r, salaries)
	}
}

// GetByID returns a salary by ID
// Access control: HR/admin can see all, others can only see their own salary
func GetByID(log *slog.Logger, repo SalaryRepository, accessChecker EmployeeAccessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetByID"
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

		salary, err := repo.GetSalaryByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("salary not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Salary not found"))
				return
			}
			log.Error("failed to get salary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary"))
			return
		}

		// Check access permission for the salary's employee
		if !authz.HasAnyRole(claims, "hr", "admin") {
			canAccess, err := authz.CanAccessSalaryData(r.Context(), claims, salary.EmployeeID, accessChecker)
			if err != nil {
				log.Error("failed to check access permission", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to verify access"))
				return
			}
			if !canAccess {
				log.Warn("access denied to salary record", slog.Int64("salary_id", id), slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, resp.Forbidden("Access denied to this salary record"))
				return
			}
		}

		log.Info("successfully retrieved salary", slog.Int64("id", salary.ID))
		render.JSON(w, r, salary)
	}
}

// Add creates a new salary record
func Add(log *slog.Logger, repo SalaryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.Add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddSalaryRequest
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

		id, err := repo.AddSalary(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate salary record")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Salary record for this employee/year/month already exists"))
				return
			}
			log.Error("failed to add salary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add salary"))
			return
		}

		log.Info("salary added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// Approve approves a salary
func Approve(log *slog.Logger, repo SalaryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.Approve"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
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

		err = repo.ApproveSalary(r.Context(), id, approverID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("salary not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Salary not found"))
				return
			}
			log.Error("failed to approve salary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to approve salary"))
			return
		}

		log.Info("salary approved", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// Pay marks salary as paid
func Pay(log *slog.Logger, repo SalaryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.Pay"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.MarkSalaryPaid(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("salary not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Salary not found"))
				return
			}
			log.Error("failed to mark salary as paid", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to mark salary as paid"))
			return
		}

		log.Info("salary paid", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// Delete deletes a salary record
func Delete(log *slog.Logger, repo SalaryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.Delete"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteSalary(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("salary not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Salary not found"))
				return
			}
			log.Error("failed to delete salary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete salary"))
			return
		}

		log.Info("salary deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Bonus Handlers ---

// GetBonuses returns bonuses with filters
func GetBonuses(log *slog.Logger, repo BonusRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetBonuses"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.BonusFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err == nil {
				filter.EmployeeID = &val
			}
		}

		if salaryIDStr := q.Get("salary_id"); salaryIDStr != "" {
			val, err := strconv.ParseInt(salaryIDStr, 10, 64)
			if err == nil {
				filter.SalaryID = &val
			}
		}

		if bonusType := q.Get("bonus_type"); bonusType != "" {
			filter.BonusType = &bonusType
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

		bonuses, err := repo.GetBonuses(r.Context(), filter)
		if err != nil {
			log.Error("failed to get bonuses", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve bonuses"))
			return
		}

		log.Info("successfully retrieved bonuses", slog.Int("count", len(bonuses)))
		render.JSON(w, r, bonuses)
	}
}

// AddBonus creates a new bonus
func AddBonus(log *slog.Logger, repo BonusRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.AddBonus"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddBonusRequest
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

		id, err := repo.AddBonus(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id or salary_id"))
				return
			}
			log.Error("failed to add bonus", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add bonus"))
			return
		}

		log.Info("bonus added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// ApproveBonus approves a bonus
func ApproveBonus(log *slog.Logger, repo BonusRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.ApproveBonus"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
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

		err = repo.ApproveBonus(r.Context(), id, approverID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("bonus not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Bonus not found"))
				return
			}
			log.Error("failed to approve bonus", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to approve bonus"))
			return
		}

		log.Info("bonus approved", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteBonus deletes a bonus
func DeleteBonus(log *slog.Logger, repo BonusRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.DeleteBonus"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteBonus(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("bonus not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Bonus not found"))
				return
			}
			log.Error("failed to delete bonus", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete bonus"))
			return
		}

		log.Info("bonus deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Deduction Handlers ---

// GetDeductions returns deductions with filters
func GetDeductions(log *slog.Logger, repo DeductionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.GetDeductions"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.DeductionFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err == nil {
				filter.EmployeeID = &val
			}
		}

		if salaryIDStr := q.Get("salary_id"); salaryIDStr != "" {
			val, err := strconv.ParseInt(salaryIDStr, 10, 64)
			if err == nil {
				filter.SalaryID = &val
			}
		}

		if deductionType := q.Get("deduction_type"); deductionType != "" {
			filter.DeductionType = &deductionType
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

		deductions, err := repo.GetDeductions(r.Context(), filter)
		if err != nil {
			log.Error("failed to get deductions", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve deductions"))
			return
		}

		log.Info("successfully retrieved deductions", slog.Int("count", len(deductions)))
		render.JSON(w, r, deductions)
	}
}

// AddDeduction creates a new deduction
func AddDeduction(log *slog.Logger, repo DeductionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.AddDeduction"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddDeductionRequest
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

		id, err := repo.AddDeduction(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id or salary_id"))
				return
			}
			log.Error("failed to add deduction", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add deduction"))
			return
		}

		log.Info("deduction added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// DeleteDeduction deletes a deduction
func DeleteDeduction(log *slog.Logger, repo DeductionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.salary.DeleteDeduction"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteDeduction(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("deduction not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Deduction not found"))
				return
			}
			log.Error("failed to delete deduction", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete deduction"))
			return
		}

		log.Info("deduction deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
