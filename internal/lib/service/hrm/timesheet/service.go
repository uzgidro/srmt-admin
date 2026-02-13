package timesheet

import (
	"context"
	"fmt"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/timesheet"
	"srmt-admin/internal/storage"
	"time"
)

type RepoInterface interface {
	// Timesheet entries
	GetTimesheetEntries(ctx context.Context, employeeID int64, year, month int) ([]*timesheet.Day, error)
	GetTimesheetEntry(ctx context.Context, id int64) (*timesheet.Day, error)
	GetTimesheetEntryByEmployeeDate(ctx context.Context, employeeID int64, date string) (*timesheet.Day, error)
	UpsertTimesheetEntry(ctx context.Context, employeeID int64, date, status string, checkIn, checkOut *string, hoursWorked, overtime *float64, isWeekend, isHoliday bool, note *string) (int64, error)
	UpdateTimesheetEntry(ctx context.Context, id int64, req dto.UpdateTimesheetEntryRequest) error
	GetEmployeesForTimesheet(ctx context.Context, filters dto.TimesheetFilters) ([]*timesheet.EmployeeInfo, error)

	// Holidays
	GetHolidays(ctx context.Context, year int) ([]*timesheet.Holiday, error)
	GetHolidaysByMonth(ctx context.Context, year, month int) ([]*timesheet.Holiday, error)
	CreateHoliday(ctx context.Context, req dto.CreateHolidayRequest) (int64, error)
	DeleteHoliday(ctx context.Context, id int64) error

	// Corrections
	GetTimesheetCorrections(ctx context.Context, filters dto.CorrectionFilters) ([]*timesheet.Correction, error)
	GetTimesheetCorrectionByID(ctx context.Context, id int64) (*timesheet.Correction, error)
	CreateTimesheetCorrection(ctx context.Context, req dto.CreateTimesheetCorrectionRequest, originalStatus, originalCheckIn, originalCheckOut *string, requestedBy int64) (int64, error)
	ApproveTimesheetCorrection(ctx context.Context, id int64, approvedBy int64) error
	RejectTimesheetCorrection(ctx context.Context, id int64, approvedBy int64, reason string) error
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// GetTimesheet returns timesheets for all matching employees for a given month.
// It generates a full array of days for the month, filling in existing entries, holidays, and weekends.
func (s *Service) GetTimesheet(ctx context.Context, filters dto.TimesheetFilters) ([]*timesheet.EmployeeTimesheet, error) {
	// 1. Get list of employees
	employees, err := s.repo.GetEmployeesForTimesheet(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("get employees: %w", err)
	}

	// 2. Get holidays for the month
	holidays, err := s.repo.GetHolidaysByMonth(ctx, filters.Year, filters.Month)
	if err != nil {
		return nil, fmt.Errorf("get holidays: %w", err)
	}
	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		holidayMap[h.Date] = true
	}

	// 3. Build timesheet for each employee
	var result []*timesheet.EmployeeTimesheet
	for _, emp := range employees {
		// Get existing entries
		entries, err := s.repo.GetTimesheetEntries(ctx, emp.EmployeeID, filters.Year, filters.Month)
		if err != nil {
			return nil, fmt.Errorf("get entries for employee %d: %w", emp.EmployeeID, err)
		}

		entryMap := make(map[string]*timesheet.Day)
		for _, e := range entries {
			entryMap[e.Date] = e
		}

		// Generate full month of days
		days, summary := s.generateMonthDays(filters.Year, filters.Month, entryMap, holidayMap)

		result = append(result, &timesheet.EmployeeTimesheet{
			EmployeeID:   emp.EmployeeID,
			EmployeeName: emp.Name,
			Department:   emp.Department,
			Position:     emp.Position,
			TabNumber:    emp.TabNumber,
			Days:         days,
			Summary:      summary,
		})
	}

	if result == nil {
		result = []*timesheet.EmployeeTimesheet{}
	}

	return result, nil
}

// generateMonthDays builds the full array of days for a month with summary
func (s *Service) generateMonthDays(year, month int, entryMap map[string]*timesheet.Day, holidayMap map[string]bool) ([]timesheet.Day, timesheet.Summary) {
	loc := time.UTC
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	lastDay := firstDay.AddDate(0, 1, -1)

	var days []timesheet.Day
	var summary timesheet.Summary

	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		isWeekend := d.Weekday() == time.Sunday
		isHoliday := holidayMap[dateStr]

		day := timesheet.Day{
			Date:      dateStr,
			IsWeekend: isWeekend,
			IsHoliday: isHoliday,
		}

		// If we have an existing entry, use it
		if entry, ok := entryMap[dateStr]; ok {
			day.ID = entry.ID
			day.EmployeeID = entry.EmployeeID
			day.Status = entry.Status
			day.CheckIn = entry.CheckIn
			day.CheckOut = entry.CheckOut
			day.HoursWorked = entry.HoursWorked
			day.Overtime = entry.Overtime
			day.Note = entry.Note
			day.IsWeekend = entry.IsWeekend
			day.IsHoliday = entry.IsHoliday
		} else {
			// Determine default status
			if isHoliday {
				day.Status = "holiday"
			} else if isWeekend {
				day.Status = "day_off"
			} else {
				// Future or today: present by default if workday
				day.Status = "present"
			}
		}

		// Update summary
		if !isWeekend && !isHoliday {
			summary.TotalWorkDays++
		}

		switch day.Status {
		case "present":
			if !isWeekend && !isHoliday {
				summary.PresentDays++
			}
		case "absent", "unauthorized":
			summary.AbsentDays++
		case "vacation":
			summary.VacationDays++
		case "sick_leave":
			summary.SickDays++
		case "business_trip":
			summary.BusinessTripDays++
		case "remote":
			summary.RemoteDays++
		}

		if day.HoursWorked != nil {
			summary.TotalHours += *day.HoursWorked
		}
		if day.Overtime != nil {
			summary.OvertimeHours += *day.Overtime
		}

		days = append(days, day)
	}

	return days, summary
}

// UpdateEntry updates an existing timesheet entry
func (s *Service) UpdateEntry(ctx context.Context, id int64, req dto.UpdateTimesheetEntryRequest) error {
	return s.repo.UpdateTimesheetEntry(ctx, id, req)
}

// GetHolidays returns holidays for a year
func (s *Service) GetHolidays(ctx context.Context, year int) ([]*timesheet.Holiday, error) {
	holidays, err := s.repo.GetHolidays(ctx, year)
	if err != nil {
		return nil, err
	}
	if holidays == nil {
		holidays = []*timesheet.Holiday{}
	}
	return holidays, nil
}

// CreateHoliday creates a new holiday
func (s *Service) CreateHoliday(ctx context.Context, req dto.CreateHolidayRequest) (int64, error) {
	return s.repo.CreateHoliday(ctx, req)
}

// DeleteHoliday deletes a holiday
func (s *Service) DeleteHoliday(ctx context.Context, id int64) error {
	return s.repo.DeleteHoliday(ctx, id)
}

// GetCorrections returns timesheet corrections
func (s *Service) GetCorrections(ctx context.Context, filters dto.CorrectionFilters) ([]*timesheet.Correction, error) {
	corrections, err := s.repo.GetTimesheetCorrections(ctx, filters)
	if err != nil {
		return nil, err
	}
	if corrections == nil {
		corrections = []*timesheet.Correction{}
	}
	return corrections, nil
}

// CreateCorrection creates a new correction, auto-populating original values
func (s *Service) CreateCorrection(ctx context.Context, req dto.CreateTimesheetCorrectionRequest, requestedBy int64) (int64, error) {
	// Try to get the current entry to populate original values
	var originalStatus, originalCheckIn, originalCheckOut *string

	existing, err := s.repo.GetTimesheetEntryByEmployeeDate(ctx, req.EmployeeID, req.Date)
	if err != nil && err != storage.ErrTimesheetEntryNotFound {
		return 0, fmt.Errorf("get existing entry: %w", err)
	}
	if existing != nil {
		originalStatus = &existing.Status
		originalCheckIn = existing.CheckIn
		originalCheckOut = existing.CheckOut
	}

	return s.repo.CreateTimesheetCorrection(ctx, req, originalStatus, originalCheckIn, originalCheckOut, requestedBy)
}

// ApproveCorrection approves a correction and applies changes to the timesheet entry
func (s *Service) ApproveCorrection(ctx context.Context, id int64, approvedBy int64) error {
	// Get the correction
	cor, err := s.repo.GetTimesheetCorrectionByID(ctx, id)
	if err != nil {
		return err
	}

	if cor.Status != "pending" {
		return storage.ErrInvalidStatus
	}

	// Approve the correction
	if err := s.repo.ApproveTimesheetCorrection(ctx, id, approvedBy); err != nil {
		return err
	}

	// Apply the correction to timesheet_entries via upsert
	dateStr := cor.Date
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("parse date: %w", err)
	}

	isWeekend := date.Weekday() == time.Sunday
	// We can't easily check holiday here without an extra query, so default to false
	// The actual holiday flag will be set if there was an existing entry
	isHoliday := false

	_, err = s.repo.UpsertTimesheetEntry(ctx, cor.EmployeeID, dateStr, cor.NewStatus,
		cor.NewCheckIn, cor.NewCheckOut, nil, nil, isWeekend, isHoliday, nil)
	if err != nil {
		s.log.Error("failed to apply correction to timesheet entry", "error", err, "correction_id", id)
		return fmt.Errorf("apply correction: %w", err)
	}

	return nil
}

// RejectCorrection rejects a correction
func (s *Service) RejectCorrection(ctx context.Context, id int64, approvedBy int64, reason string) error {
	cor, err := s.repo.GetTimesheetCorrectionByID(ctx, id)
	if err != nil {
		return err
	}

	if cor.Status != "pending" {
		return storage.ErrInvalidStatus
	}

	return s.repo.RejectTimesheetCorrection(ctx, id, approvedBy, reason)
}
