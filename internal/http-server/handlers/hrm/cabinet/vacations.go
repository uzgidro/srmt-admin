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
	"github.com/go-playground/validator/v10"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// VacationRepository defines the interface for vacation operations
type VacationRepository interface {
	GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error)
	GetVacations(ctx context.Context, filter hrm.VacationFilter) ([]*hrmmodel.Vacation, error)
	AddVacation(ctx context.Context, req hrm.AddVacationRequest) (int64, error)
	GetVacationByID(ctx context.Context, id int64) (*hrmmodel.Vacation, error)
	CancelVacation(ctx context.Context, id int64) error
}

// GetMyVacations returns vacation requests for the currently authenticated employee
func GetMyVacations(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMyVacations"
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

		// Get vacations for the employee
		vacations, err := repo.GetVacations(r.Context(), hrm.VacationFilter{
			EmployeeID: &employee.ID,
			Limit:      50,
		})
		if err != nil {
			log.Error("failed to get vacations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacations"))
			return
		}

		// Convert to response format
		response := make([]hrm.MyVacationResponse, 0, len(vacations))
		for _, v := range vacations {
			item := hrm.MyVacationResponse{
				ID:              v.ID,
				VacationTypeID:  v.VacationTypeID,
				StartDate:       v.StartDate,
				EndDate:         v.EndDate,
				DaysCount:       v.DaysCount,
				Status:          v.Status,
				Reason:          v.Reason,
				RejectionReason: v.RejectionReason,
				RequestedAt:     v.RequestedAt,
				ApprovedAt:      v.ApprovedAt,
				SubstituteID:    v.SubstituteEmployeeID,
			}

			if v.VacationType != nil {
				item.VacationTypeName = v.VacationType.Name
			}

			response = append(response, item)
		}

		log.Info("vacations retrieved", slog.Int64("employee_id", employee.ID), slog.Int("count", len(vacations)))
		render.JSON(w, r, response)
	}
}

// CreateMyVacation creates a new vacation request for the currently authenticated employee
func CreateMyVacation(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.CreateMyVacation"
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

		// Parse request
		var req hrm.MyVacationRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Warn("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request body"))
			return
		}

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			validationErrs := err.(validator.ValidationErrors)
			log.Warn("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(validationErrs))
			return
		}

		// Parse dates
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			log.Warn("invalid start_date format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid start_date format, expected YYYY-MM-DD"))
			return
		}

		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			log.Warn("invalid end_date format", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid end_date format, expected YYYY-MM-DD"))
			return
		}

		// Validate date range
		if endDate.Before(startDate) {
			log.Warn("end_date before start_date")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_date cannot be before start_date"))
			return
		}

		// Calculate days count (simplified - should account for weekends/holidays)
		daysCount := float64(endDate.Sub(startDate).Hours()/24) + 1

		// Create vacation request
		addReq := hrm.AddVacationRequest{
			EmployeeID:           employee.ID,
			VacationTypeID:       int(req.VacationTypeID),
			StartDate:            startDate,
			EndDate:              endDate,
			DaysCount:            daysCount,
			Reason:               req.Reason,
			SubstituteEmployeeID: req.SubstituteID,
		}

		id, err := repo.AddVacation(r.Context(), addReq)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("invalid vacation type or substitute", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid vacation type or substitute employee"))
				return
			}
			log.Error("failed to create vacation request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create vacation request"))
			return
		}

		log.Info("vacation request created", slog.Int64("employee_id", employee.ID), slog.Int64("vacation_id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]int64{"id": id})
	}
}

// CancelMyVacation cancels a vacation request for the currently authenticated employee
func CancelMyVacation(log *slog.Logger, repo VacationRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.CancelMyVacation"
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

		// Get vacation ID from URL
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid id parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		// Get vacation to verify ownership
		vacation, err := repo.GetVacationByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacation request not found"))
				return
			}
			log.Error("failed to get vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacation"))
			return
		}

		// Verify ownership
		if vacation.EmployeeID != employee.ID {
			log.Warn("vacation does not belong to employee", slog.Int64("vacation_id", id), slog.Int64("employee_id", employee.ID))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, resp.Forbidden("You can only cancel your own vacation requests"))
			return
		}

		// Cancel vacation
		if err := repo.CancelVacation(r.Context(), id); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacation cannot be cancelled", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Vacation cannot be cancelled (already cancelled or taken)"))
				return
			}
			log.Error("failed to cancel vacation", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to cancel vacation"))
			return
		}

		log.Info("vacation cancelled", slog.Int64("employee_id", employee.ID), slog.Int64("vacation_id", id))
		render.JSON(w, r, resp.OK())
	}
}
