package cabinet

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// LeaveBalanceRepository defines the interface for leave balance operations
type LeaveBalanceRepository interface {
	GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error)
	GetVacationBalances(ctx context.Context, filter hrm.VacationBalanceFilter) ([]*hrmmodel.VacationBalance, error)
}

// GetLeaveBalance returns the leave balance for the currently authenticated employee
func GetLeaveBalance(log *slog.Logger, repo LeaveBalanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetLeaveBalance"
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

		// Get current year
		currentYear := time.Now().Year()

		// Get vacation balances for the employee
		balances, err := repo.GetVacationBalances(r.Context(), hrm.VacationBalanceFilter{
			EmployeeID: &employee.ID,
			Year:       &currentYear,
		})
		if err != nil {
			log.Error("failed to get vacation balances", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve leave balance"))
			return
		}

		// Build response
		response := hrm.MyLeaveBalanceResponse{
			EmployeeID: employee.ID,
			Year:       currentYear,
			Balances:   make([]hrm.LeaveBalanceDetail, 0, len(balances)),
		}

		for _, b := range balances {
			detail := hrm.LeaveBalanceDetail{
				VacationTypeID:  b.VacationTypeID,
				EntitledDays:    b.EntitledDays,
				UsedDays:        b.UsedDays,
				CarriedOverDays: b.CarriedOverDays,
				AdjustmentDays:  b.AdjustmentDays,
				RemainingDays:   b.RemainingDays,
			}

			if b.VacationType != nil {
				detail.VacationTypeName = b.VacationType.Name
				detail.VacationTypeCode = b.VacationType.Code
			}

			response.Balances = append(response.Balances, detail)
		}

		log.Info("leave balance retrieved", slog.Int64("employee_id", employee.ID), slog.Int("balance_count", len(balances)))
		render.JSON(w, r, response)
	}
}
