package cabinet

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

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// SalaryRepository defines the interface for salary operations
type SalaryRepository interface {
	GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error)
	GetSalaries(ctx context.Context, filter hrm.SalaryFilter) ([]*hrmmodel.Salary, error)
	GetSalaryByID(ctx context.Context, id int64) (*hrmmodel.Salary, error)
	GetCurrentSalaryStructure(ctx context.Context, employeeID int64) (*hrmmodel.SalaryStructure, error)
}

// GetMySalary returns salary information for the currently authenticated employee
func GetMySalary(log *slog.Logger, repo SalaryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMySalary"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Find employee by user_id
		employee, err := repo.GetEmployeeByUserID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("employee not found", slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Employee profile not found"))
				return
			}
			log.Error("failed to get employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee"))
			return
		}

		// Get recent salaries (last 12 months)
		salaries, err := repo.GetSalaries(r.Context(), hrm.SalaryFilter{
			EmployeeID: &employee.ID,
			Limit:      12,
		})
		if err != nil {
			log.Error("failed to get salaries", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary information"))
			return
		}

		// Get current salary structure
		salaryStructure, err := repo.GetCurrentSalaryStructure(r.Context(), employee.ID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			log.Error("failed to get salary structure", sl.Err(err))
			// Non-fatal error, continue without structure
		}

		// Build response
		response := hrm.MySalaryResponse{
			RecentPayslips: make([]hrm.MyPayslipSummary, 0, len(salaries)),
		}

		// Current salary is the most recent one
		if len(salaries) > 0 {
			current := salaries[0]
			response.CurrentSalary = &hrm.MySalaryDetail{
				ID:               current.ID,
				Year:             current.Year,
				Month:            current.Month,
				GrossAmount:      current.GrossAmount,
				NetAmount:        current.NetAmount,
				TaxAmount:        current.TaxAmount,
				BonusesAmount:    current.BonusesAmount,
				DeductionsAmount: current.DeductionsAmount,
				Status:           current.Status,
				PaidAt:           current.PaidAt,
			}
		}

		// Recent payslips
		for _, s := range salaries {
			response.RecentPayslips = append(response.RecentPayslips, hrm.MyPayslipSummary{
				ID:        s.ID,
				Year:      s.Year,
				Month:     s.Month,
				NetAmount: s.NetAmount,
				Status:    s.Status,
				PaidAt:    s.PaidAt,
			})
		}

		// Salary structure
		if salaryStructure != nil {
			response.SalaryStructure = &hrm.MySalaryStructure{
				BaseSalary:    salaryStructure.BaseSalary,
				Currency:      salaryStructure.Currency,
				PayFrequency:  salaryStructure.PayFrequency,
				EffectiveFrom: salaryStructure.EffectiveFrom.Format("2006-01-02"),
			}
		}

		log.Info("salary info retrieved", slog.Int64("employee_id", employee.ID), slog.Int("payslip_count", len(salaries)))
		render.JSON(w, r, response)
	}
}

// GetMyPayslip returns a specific payslip for the currently authenticated employee
func GetMyPayslip(log *slog.Logger, repo SalaryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMyPayslip"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Find employee by user_id
		employee, err := repo.GetEmployeeByUserID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("employee not found", slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Employee profile not found"))
				return
			}
			log.Error("failed to get employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee"))
			return
		}

		// Get salary ID from URL
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid id parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// Get salary
		salary, err := repo.GetSalaryByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("salary not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Payslip not found"))
				return
			}
			log.Error("failed to get salary", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve payslip"))
			return
		}

		// Verify ownership
		if salary.EmployeeID != employee.ID {
			log.Warn("salary does not belong to employee", slog.Int64("salary_id", id), slog.Int64("employee_id", employee.ID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("You can only access your own payslips"))
			return
		}

		// Build response
		response := struct {
			ID               int64      `json:"id"`
			Year             int        `json:"year"`
			Month            int        `json:"month"`
			Period           string     `json:"period"`
			BaseAmount       float64    `json:"base_amount"`
			AllowancesAmount float64    `json:"allowances_amount"`
			BonusesAmount    float64    `json:"bonuses_amount"`
			DeductionsAmount float64    `json:"deductions_amount"`
			GrossAmount      float64    `json:"gross_amount"`
			TaxAmount        float64    `json:"tax_amount"`
			NetAmount        float64    `json:"net_amount"`
			WorkedDays       int        `json:"worked_days"`
			TotalWorkDays    int        `json:"total_work_days"`
			OvertimeHours    float64    `json:"overtime_hours"`
			Status           string     `json:"status"`
			PaidAt           *time.Time `json:"paid_at,omitempty"`
		}{
			ID:               salary.ID,
			Year:             salary.Year,
			Month:            salary.Month,
			Period:           time.Month(salary.Month).String() + " " + strconv.Itoa(salary.Year),
			BaseAmount:       salary.BaseAmount,
			AllowancesAmount: salary.AllowancesAmount,
			BonusesAmount:    salary.BonusesAmount,
			DeductionsAmount: salary.DeductionsAmount,
			GrossAmount:      salary.GrossAmount,
			TaxAmount:        salary.TaxAmount,
			NetAmount:        salary.NetAmount,
			WorkedDays:       salary.WorkedDays,
			TotalWorkDays:    salary.TotalWorkDays,
			OvertimeHours:    salary.OvertimeHours,
			Status:           salary.Status,
			PaidAt:           salary.PaidAt,
		}

		log.Info("payslip retrieved", slog.Int64("employee_id", employee.ID), slog.Int64("salary_id", id))
		render.JSON(w, r, response)
	}
}
