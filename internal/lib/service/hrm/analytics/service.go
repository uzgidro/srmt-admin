package analytics

import (
	"context"
	"log/slog"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/hrm/analytics"
)

type RepoInterface interface {
	GetAnalyticsDashboard(ctx context.Context, filter dto.ReportFilter) (*analytics.Dashboard, error)
	GetHeadcountReport(ctx context.Context, filter dto.ReportFilter) (*analytics.HeadcountReport, error)
	GetHeadcountTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.HeadcountTrend, error)
	GetTurnoverReport(ctx context.Context, filter dto.ReportFilter) (*analytics.TurnoverReport, error)
	GetTurnoverTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.TurnoverTrend, error)
	GetAttendanceReport(ctx context.Context, filter dto.ReportFilter) (*analytics.AttendanceReport, error)
	GetSalaryReport(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryReport, error)
	GetSalaryTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryTrend, error)
	GetPerformanceAnalytics(ctx context.Context, filter dto.ReportFilter) (*analytics.PerformanceAnalytics, error)
	GetTrainingAnalytics(ctx context.Context, filter dto.ReportFilter) (*analytics.TrainingAnalytics, error)
	GetDemographicsReport(ctx context.Context, filter dto.ReportFilter) (*analytics.DemographicsReport, error)
}

type Service struct {
	repo RepoInterface
	log  *slog.Logger
}

func NewService(repo RepoInterface, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) GetDashboard(ctx context.Context, filter dto.ReportFilter) (*analytics.Dashboard, error) {
	d, err := s.repo.GetAnalyticsDashboard(ctx, filter)
	if err != nil {
		return nil, err
	}
	if d.AgeDistribution == nil {
		d.AgeDistribution = []*analytics.DistributionItem{}
	}
	if d.TenureDistribution == nil {
		d.TenureDistribution = []*analytics.DistributionItem{}
	}
	if d.DepartmentHeadcount == nil {
		d.DepartmentHeadcount = []*analytics.DepartmentHeadcount{}
	}
	if d.PositionHeadcount == nil {
		d.PositionHeadcount = []*analytics.PositionHeadcount{}
	}
	return d, nil
}

func (s *Service) GetHeadcountReport(ctx context.Context, filter dto.ReportFilter) (*analytics.HeadcountReport, error) {
	r, err := s.repo.GetHeadcountReport(ctx, filter)
	if err != nil {
		return nil, err
	}
	if r.ByDepartment == nil {
		r.ByDepartment = []*analytics.DepartmentHeadcount{}
	}
	if r.ByPosition == nil {
		r.ByPosition = []*analytics.PositionHeadcount{}
	}
	return r, nil
}

func (s *Service) GetHeadcountTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.HeadcountTrend, error) {
	t, err := s.repo.GetHeadcountTrend(ctx, filter)
	if err != nil {
		return nil, err
	}
	if t.Points == nil {
		t.Points = []*analytics.TrendPoint{}
	}
	return t, nil
}

func (s *Service) GetTurnoverReport(ctx context.Context, filter dto.ReportFilter) (*analytics.TurnoverReport, error) {
	r, err := s.repo.GetTurnoverReport(ctx, filter)
	if err != nil {
		return nil, err
	}
	if r.ByReason == nil {
		r.ByReason = []*analytics.DistributionItem{}
	}
	if r.ByDepartment == nil {
		r.ByDepartment = []*analytics.DepartmentTurnover{}
	}
	return r, nil
}

func (s *Service) GetTurnoverTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.TurnoverTrend, error) {
	t, err := s.repo.GetTurnoverTrend(ctx, filter)
	if err != nil {
		return nil, err
	}
	if t.Points == nil {
		t.Points = []*analytics.TrendPoint{}
	}
	return t, nil
}

func (s *Service) GetAttendanceReport(ctx context.Context, filter dto.ReportFilter) (*analytics.AttendanceReport, error) {
	r, err := s.repo.GetAttendanceReport(ctx, filter)
	if err != nil {
		return nil, err
	}
	if r.ByStatus == nil {
		r.ByStatus = []*analytics.DistributionItem{}
	}
	if r.ByDepartment == nil {
		r.ByDepartment = []*analytics.DepartmentAttendance{}
	}
	return r, nil
}

func (s *Service) GetSalaryReport(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryReport, error) {
	r, err := s.repo.GetSalaryReport(ctx, filter)
	if err != nil {
		return nil, err
	}
	if r.ByDepartment == nil {
		r.ByDepartment = []*analytics.DepartmentSalary{}
	}
	return r, nil
}

func (s *Service) GetSalaryTrend(ctx context.Context, filter dto.ReportFilter) (*analytics.SalaryTrend, error) {
	t, err := s.repo.GetSalaryTrend(ctx, filter)
	if err != nil {
		return nil, err
	}
	if t.Points == nil {
		t.Points = []*analytics.TrendPoint{}
	}
	return t, nil
}

func (s *Service) GetPerformanceAnalytics(ctx context.Context, filter dto.ReportFilter) (*analytics.PerformanceAnalytics, error) {
	r, err := s.repo.GetPerformanceAnalytics(ctx, filter)
	if err != nil {
		return nil, err
	}
	if r.RatingDistribution == nil {
		r.RatingDistribution = []*analytics.DistributionItem{}
	}
	if r.ByDepartment == nil {
		r.ByDepartment = []*analytics.DepartmentPerformance{}
	}
	return r, nil
}

func (s *Service) GetTrainingAnalytics(ctx context.Context, filter dto.ReportFilter) (*analytics.TrainingAnalytics, error) {
	r, err := s.repo.GetTrainingAnalytics(ctx, filter)
	if err != nil {
		return nil, err
	}
	if r.ByStatus == nil {
		r.ByStatus = []*analytics.DistributionItem{}
	}
	if r.ByType == nil {
		r.ByType = []*analytics.DistributionItem{}
	}
	return r, nil
}

func (s *Service) GetDemographicsReport(ctx context.Context, filter dto.ReportFilter) (*analytics.DemographicsReport, error) {
	r, err := s.repo.GetDemographicsReport(ctx, filter)
	if err != nil {
		return nil, err
	}
	if r.AgeDistribution == nil {
		r.AgeDistribution = []*analytics.DistributionItem{}
	}
	if r.TenureDistribution == nil {
		r.TenureDistribution = []*analytics.DistributionItem{}
	}
	return r, nil
}

func (s *Service) GetDiversityReport(_ context.Context, _ dto.ReportFilter) (*analytics.DiversityReport, error) {
	return &analytics.DiversityReport{
		Message: "Diversity data is not available. The system does not collect ethnicity, nationality or other diversity attributes.",
	}, nil
}

func (s *Service) GetCustomReport(ctx context.Context, filter dto.ReportFilter) (interface{}, error) {
	if filter.ReportType == nil {
		return s.GetDashboard(ctx, filter)
	}
	switch *filter.ReportType {
	case "headcount":
		return s.GetHeadcountReport(ctx, filter)
	case "turnover":
		return s.GetTurnoverReport(ctx, filter)
	case "attendance":
		return s.GetAttendanceReport(ctx, filter)
	case "salary":
		return s.GetSalaryReport(ctx, filter)
	case "performance":
		return s.GetPerformanceAnalytics(ctx, filter)
	case "training":
		return s.GetTrainingAnalytics(ctx, filter)
	case "demographics":
		return s.GetDemographicsReport(ctx, filter)
	default:
		return s.GetDashboard(ctx, filter)
	}
}
