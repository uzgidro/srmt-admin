package vacation

import (
	"context"
	"fmt"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/vacation"
	"srmt-admin/internal/storage"
	"time"
)

// BalanceRequiredTypes are vacation types that require balance checking
var BalanceRequiredTypes = map[string]bool{
	"annual":     true,
	"additional": true,
	"study":      true,
	"comp":       true,
}

type RepoInterface interface {
	CreateVacation(ctx context.Context, req dto.CreateVacationRequest, days int, createdBy int64) (int64, error)
	GetVacationByID(ctx context.Context, id int64) (*vacation.Vacation, error)
	GetAllVacations(ctx context.Context, filters dto.VacationFilters) ([]*vacation.Vacation, error)
	UpdateVacation(ctx context.Context, id int64, req dto.EditVacationRequest, days *int) error
	DeleteVacation(ctx context.Context, id int64) error
	UpdateVacationStatus(ctx context.Context, id int64, status string) error
	ApproveVacation(ctx context.Context, id int64, approvedBy int64) error
	RejectVacation(ctx context.Context, id int64, rejectedBy int64, reason string) error
	CheckVacationOverlap(ctx context.Context, employeeID int64, startDate, endDate string, excludeID *int64) (bool, error)
	CheckBlockedPeriod(ctx context.Context, departmentID int64, startDate, endDate string) (bool, error)
	GetVacationBalance(ctx context.Context, employeeID int64, year int) (*vacation.Balance, error)
	GetAllVacationBalances(ctx context.Context, year int) ([]*vacation.Balance, error)
	GetPendingVacations(ctx context.Context) ([]*vacation.Vacation, error)
	GetVacationCalendar(ctx context.Context, filters dto.VacationCalendarFilters) ([]*vacation.CalendarEntry, error)
	UpdateVacationBalancePending(ctx context.Context, employeeID int64, year int, deltaPending int) error
	UpdateVacationBalanceApprove(ctx context.Context, employeeID int64, year int, days int) error
	UpdateVacationBalanceReject(ctx context.Context, employeeID int64, year int, days int) error
	UpdateVacationBalanceCancelApproved(ctx context.Context, employeeID int64, year int, days int) error
	GetEmployeeDepartmentID(ctx context.Context, employeeID int64) (int64, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) Create(ctx context.Context, req dto.CreateVacationRequest, createdBy int64) (int64, error) {
	// 1. Validate dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return 0, storage.ErrInvalidDateRange
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return 0, storage.ErrInvalidDateRange
	}

	today := time.Now().Truncate(24 * time.Hour)
	if startDate.Before(today) {
		return 0, storage.ErrStartDateInPast
	}
	if endDate.Before(startDate) {
		return 0, storage.ErrInvalidDateRange
	}

	// 2. Calculate business days (Mon-Sat for Uzbekistan, Sunday is day off)
	days := calculateBusinessDays(startDate, endDate)

	// 3. Balance check for required types
	year := startDate.Year()
	if BalanceRequiredTypes[req.VacationType] {
		balance, err := s.repo.GetVacationBalance(ctx, req.EmployeeID, year)
		if err != nil {
			return 0, fmt.Errorf("failed to get balance: %w", err)
		}
		if balance.RemainingDays < days {
			return 0, storage.ErrInsufficientBalance
		}
	}

	// 4. Overlap detection
	overlaps, err := s.repo.CheckVacationOverlap(ctx, req.EmployeeID, req.StartDate, req.EndDate, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to check overlap: %w", err)
	}
	if overlaps {
		return 0, storage.ErrVacationOverlap
	}

	// 5. Blocked period check
	deptID, err := s.repo.GetEmployeeDepartmentID(ctx, req.EmployeeID)
	if err != nil {
		return 0, fmt.Errorf("failed to get department: %w", err)
	}
	if deptID > 0 {
		blocked, err := s.repo.CheckBlockedPeriod(ctx, deptID, req.StartDate, req.EndDate)
		if err != nil {
			return 0, fmt.Errorf("failed to check blocked period: %w", err)
		}
		if blocked {
			return 0, storage.ErrBlockedPeriod
		}
	}

	// 6. Create vacation
	id, err := s.repo.CreateVacation(ctx, req, days, createdBy)
	if err != nil {
		return 0, err
	}

	// 7. Update balance: add pending days
	if BalanceRequiredTypes[req.VacationType] {
		if err := s.repo.UpdateVacationBalancePending(ctx, req.EmployeeID, year, days); err != nil {
			s.log.Error("failed to update balance pending", "error", err, "vacation_id", id)
		}
	}

	return id, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*vacation.Vacation, error) {
	return s.repo.GetVacationByID(ctx, id)
}

func (s *Service) GetAll(ctx context.Context, filters dto.VacationFilters) ([]*vacation.Vacation, error) {
	return s.repo.GetAllVacations(ctx, filters)
}

func (s *Service) Update(ctx context.Context, id int64, req dto.EditVacationRequest) error {
	vac, err := s.repo.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}

	// Only draft/pending can be updated
	if vac.Status != "draft" && vac.Status != "pending" {
		return storage.ErrInvalidStatus
	}

	// Recalculate days if dates changed
	var days *int
	startStr := vac.StartDate
	endStr := vac.EndDate
	if req.StartDate != nil {
		startStr = *req.StartDate
	}
	if req.EndDate != nil {
		endStr = *req.EndDate
	}
	if req.StartDate != nil || req.EndDate != nil {
		start, _ := time.Parse("2006-01-02", startStr)
		end, _ := time.Parse("2006-01-02", endStr)
		d := calculateBusinessDays(start, end)
		days = &d
	}

	return s.repo.UpdateVacation(ctx, id, req, days)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	vac, err := s.repo.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}

	// Only draft can be deleted
	if vac.Status != "draft" {
		return storage.ErrInvalidStatus
	}

	return s.repo.DeleteVacation(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id int64, approvedBy int64) error {
	vac, err := s.repo.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}

	if vac.Status != "pending" {
		return storage.ErrInvalidStatus
	}

	if err := s.repo.ApproveVacation(ctx, id, approvedBy); err != nil {
		return err
	}

	// Update balance: pending -> used
	year, _ := time.Parse("2006-01-02", vac.StartDate)
	if BalanceRequiredTypes[vac.VacationType] {
		if err := s.repo.UpdateVacationBalanceApprove(ctx, vac.EmployeeID, year.Year(), vac.Days); err != nil {
			s.log.Error("failed to update balance on approve", "error", err, "vacation_id", id)
		}
	}

	return nil
}

func (s *Service) Reject(ctx context.Context, id int64, rejectedBy int64, reason string) error {
	vac, err := s.repo.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}

	if vac.Status != "pending" {
		return storage.ErrInvalidStatus
	}

	if err := s.repo.RejectVacation(ctx, id, rejectedBy, reason); err != nil {
		return err
	}

	// Update balance: remove pending
	year, _ := time.Parse("2006-01-02", vac.StartDate)
	if BalanceRequiredTypes[vac.VacationType] {
		if err := s.repo.UpdateVacationBalanceReject(ctx, vac.EmployeeID, year.Year(), vac.Days); err != nil {
			s.log.Error("failed to update balance on reject", "error", err, "vacation_id", id)
		}
	}

	return nil
}

func (s *Service) Cancel(ctx context.Context, id int64) error {
	vac, err := s.repo.GetVacationByID(ctx, id)
	if err != nil {
		return err
	}

	// draft, pending, approved can be cancelled
	switch vac.Status {
	case "draft", "pending", "approved":
		// OK
	default:
		return storage.ErrInvalidStatus
	}

	wasApproved := vac.Status == "approved"

	if err := s.repo.UpdateVacationStatus(ctx, id, "cancelled"); err != nil {
		return err
	}

	year, _ := time.Parse("2006-01-02", vac.StartDate)
	if BalanceRequiredTypes[vac.VacationType] {
		if vac.Status == "pending" {
			// Remove pending days
			if err := s.repo.UpdateVacationBalanceReject(ctx, vac.EmployeeID, year.Year(), vac.Days); err != nil {
				s.log.Error("failed to update balance on cancel pending", "error", err, "vacation_id", id)
			}
		} else if wasApproved {
			// Remove used days
			if err := s.repo.UpdateVacationBalanceCancelApproved(ctx, vac.EmployeeID, year.Year(), vac.Days); err != nil {
				s.log.Error("failed to update balance on cancel approved", "error", err, "vacation_id", id)
			}
		}
	}

	return nil
}

func (s *Service) GetBalance(ctx context.Context, employeeID int64, year int) (*vacation.Balance, error) {
	return s.repo.GetVacationBalance(ctx, employeeID, year)
}

func (s *Service) GetAllBalances(ctx context.Context, year int) ([]*vacation.Balance, error) {
	return s.repo.GetAllVacationBalances(ctx, year)
}

func (s *Service) GetPending(ctx context.Context) ([]*vacation.Vacation, error) {
	return s.repo.GetPendingVacations(ctx)
}

func (s *Service) GetCalendar(ctx context.Context, filters dto.VacationCalendarFilters) ([]*vacation.CalendarEntry, error) {
	return s.repo.GetVacationCalendar(ctx, filters)
}

// calculateBusinessDays counts Mon-Sat days (Uzbekistan work week: Mon-Sat)
func calculateBusinessDays(start, end time.Time) int {
	days := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Sunday {
			days++
		}
	}
	return days
}
