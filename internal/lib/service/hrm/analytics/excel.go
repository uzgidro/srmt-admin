package analytics

import (
	"context"
	"fmt"
	"srmt-admin/internal/lib/dto"

	"github.com/xuri/excelize/v2"
)

func (s *Service) ExportExcel(ctx context.Context, filter dto.ReportFilter) (*excelize.File, error) {
	reportType := "dashboard"
	if filter.ReportType != nil {
		reportType = *filter.ReportType
	}

	f := excelize.NewFile()
	sheet := "Report"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#DAEEF3"}, Pattern: 1},
	})

	switch reportType {
	case "headcount":
		return s.exportHeadcount(ctx, f, sheet, headerStyle, filter)
	case "turnover":
		return s.exportTurnover(ctx, f, sheet, headerStyle, filter)
	case "attendance":
		return s.exportAttendance(ctx, f, sheet, headerStyle, filter)
	case "salary":
		return s.exportSalary(ctx, f, sheet, headerStyle, filter)
	case "performance":
		return s.exportPerformance(ctx, f, sheet, headerStyle, filter)
	case "training":
		return s.exportTraining(ctx, f, sheet, headerStyle, filter)
	case "demographics":
		return s.exportDemographics(ctx, f, sheet, headerStyle, filter)
	default:
		return s.exportDashboard(ctx, f, sheet, headerStyle, filter)
	}
}

func (s *Service) exportDashboard(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetDashboard(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	f.SetCellStyle(sheet, "A1", "B1", style)

	rows := [][]interface{}{
		{"Total Employees", data.TotalEmployees},
		{"New Hires (Month)", data.NewHiresMonth},
		{"Terminations (Month)", data.TerminationsMonth},
		{"Turnover Rate (%)", data.TurnoverRate},
		{"Avg Tenure (Years)", data.AvgTenureYears},
		{"Avg Age", data.AvgAge},
	}
	for i, row := range rows {
		cell := fmt.Sprintf("A%d", i+2)
		f.SetCellValue(sheet, cell, row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), row[1])
		_ = cell
	}

	if len(data.DepartmentHeadcount) > 0 {
		deptSheet := "Departments"
		f.NewSheet(deptSheet)
		f.SetCellValue(deptSheet, "A1", "Department")
		f.SetCellValue(deptSheet, "B1", "Headcount")
		f.SetCellStyle(deptSheet, "A1", "B1", style)
		for i, d := range data.DepartmentHeadcount {
			f.SetCellValue(deptSheet, fmt.Sprintf("A%d", i+2), d.DepartmentName)
			f.SetCellValue(deptSheet, fmt.Sprintf("B%d", i+2), d.Headcount)
		}
	}

	return f, nil
}

func (s *Service) exportHeadcount(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetHeadcountReport(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Total Employees")
	f.SetCellValue(sheet, "B1", data.TotalEmployees)
	f.SetCellStyle(sheet, "A1", "A1", style)

	f.SetCellValue(sheet, "A3", "Department")
	f.SetCellValue(sheet, "B3", "Headcount")
	f.SetCellStyle(sheet, "A3", "B3", style)
	for i, d := range data.ByDepartment {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+4), d.DepartmentName)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+4), d.Headcount)
	}

	return f, nil
}

func (s *Service) exportTurnover(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetTurnoverReport(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	f.SetCellStyle(sheet, "A1", "B1", style)

	rows := [][]interface{}{
		{"Period", data.PeriodStart + " — " + data.PeriodEnd},
		{"Total Terminations", data.TotalTerminations},
		{"Turnover Rate (%)", data.TurnoverRate},
		{"Retention Rate (%)", data.RetentionRate},
		{"Avg Tenure at Termination (Years)", data.AvgTenureAtTermination},
	}
	for i, row := range rows {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), row[1])
	}

	return f, nil
}

func (s *Service) exportAttendance(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetAttendanceReport(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	f.SetCellStyle(sheet, "A1", "B1", style)

	rows := [][]interface{}{
		{"Period", data.PeriodStart + " — " + data.PeriodEnd},
		{"Total Work Days", data.TotalWorkDays},
		{"Avg Attendance (%)", data.AvgAttendance},
		{"Avg Absence (%)", data.AvgAbsence},
	}
	for i, row := range rows {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), row[1])
	}

	if len(data.ByDepartment) > 0 {
		r := len(rows) + 3
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), "Department")
		f.SetCellValue(sheet, fmt.Sprintf("B%d", r), "Attendance %")
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), "Absence %")
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("C%d", r), style)
		for i, d := range data.ByDepartment {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", r+i+1), d.Department)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", r+i+1), d.AttendanceRate)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", r+i+1), d.AbsenceRate)
		}
	}

	return f, nil
}

func (s *Service) exportSalary(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetSalaryReport(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	f.SetCellStyle(sheet, "A1", "B1", style)

	rows := [][]interface{}{
		{"Period", data.PeriodStart + " — " + data.PeriodEnd},
		{"Total Payroll", data.TotalPayroll},
		{"Avg Salary", data.AvgSalary},
		{"Median Salary", data.MedianSalary},
		{"Min Salary", data.MinSalary},
		{"Max Salary", data.MaxSalary},
	}
	for i, row := range rows {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), row[1])
	}

	if len(data.ByDepartment) > 0 {
		r := len(rows) + 3
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), "Department")
		f.SetCellValue(sheet, fmt.Sprintf("B%d", r), "Avg Salary")
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), "Total Payroll")
		f.SetCellValue(sheet, fmt.Sprintf("D%d", r), "Headcount")
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("D%d", r), style)
		for i, d := range data.ByDepartment {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", r+i+1), d.Department)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", r+i+1), d.AvgSalary)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", r+i+1), d.TotalPayroll)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", r+i+1), d.Headcount)
		}
	}

	return f, nil
}

func (s *Service) exportPerformance(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetPerformanceAnalytics(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	f.SetCellStyle(sheet, "A1", "B1", style)

	rows := [][]interface{}{
		{"Total Reviews", data.TotalReviews},
		{"Avg Rating", data.AvgRating},
		{"Goals Total", data.GoalCompletion.Total},
		{"Goals Completed", data.GoalCompletion.Completed},
		{"Goal Completion Rate (%)", data.GoalCompletion.Rate},
	}
	for i, row := range rows {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), row[1])
	}

	return f, nil
}

func (s *Service) exportTraining(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetTrainingAnalytics(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	f.SetCellStyle(sheet, "A1", "B1", style)

	rows := [][]interface{}{
		{"Total Trainings", data.TotalTrainings},
		{"Total Participants", data.TotalParticipants},
		{"Completion Rate (%)", data.CompletionRate},
	}
	for i, row := range rows {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), row[0])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), row[1])
	}

	return f, nil
}

func (s *Service) exportDemographics(ctx context.Context, f *excelize.File, sheet string, style int, filter dto.ReportFilter) (*excelize.File, error) {
	data, err := s.GetDemographicsReport(ctx, filter)
	if err != nil {
		return nil, err
	}

	f.SetCellValue(sheet, "A1", "Metric")
	f.SetCellValue(sheet, "B1", "Value")
	f.SetCellStyle(sheet, "A1", "B1", style)

	f.SetCellValue(sheet, "A2", "Total Employees")
	f.SetCellValue(sheet, "B2", data.TotalEmployees)
	f.SetCellValue(sheet, "A3", "Avg Age")
	f.SetCellValue(sheet, "B3", data.AvgAge)

	if len(data.AgeDistribution) > 0 {
		f.SetCellValue(sheet, "A5", "Age Group")
		f.SetCellValue(sheet, "B5", "Count")
		f.SetCellValue(sheet, "C5", "Percentage")
		f.SetCellStyle(sheet, "A5", "C5", style)
		for i, d := range data.AgeDistribution {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", i+6), d.Label)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", i+6), d.Count)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", i+6), d.Percentage)
		}
	}

	return f, nil
}
