package providers

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/lib/service/alarm"
	"srmt-admin/internal/lib/service/ascue"
	hrmaccess "srmt-admin/internal/lib/service/hrm/access"
	hrmanalytics "srmt-admin/internal/lib/service/hrm/analytics"
	hrmcompetency "srmt-admin/internal/lib/service/hrm/competency"
	hrmdashboard "srmt-admin/internal/lib/service/hrm/dashboard"
	hrmdocument "srmt-admin/internal/lib/service/hrm/document"
	hrmorgstructure "srmt-admin/internal/lib/service/hrm/orgstructure"
	hrmperformance "srmt-admin/internal/lib/service/hrm/performance"
	hrmpersonnel "srmt-admin/internal/lib/service/hrm/personnel"
	hrmrecruiting "srmt-admin/internal/lib/service/hrm/recruiting"
	hrmsalary "srmt-admin/internal/lib/service/hrm/salary"
	hrmtimesheet "srmt-admin/internal/lib/service/hrm/timesheet"
	hrmtraining "srmt-admin/internal/lib/service/hrm/training"
	hrmvacation "srmt-admin/internal/lib/service/hrm/vacation"
	"srmt-admin/internal/lib/service/metrics"
	"srmt-admin/internal/lib/service/reservoir"
	reservoirhourly "srmt-admin/internal/lib/service/reservoir-hourly"
	"srmt-admin/internal/storage/redis"
	"srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
	"time"

	"github.com/google/wire"
)

// ServiceProviderSet provides all business service dependencies
var ServiceProviderSet = wire.NewSet(
	ProvideTokenService,
	ProvideASCUEFetcher,
	ProvideMetricsBlender,
	ProvideReservoirFetcher,
	ProvideHTTPClient,
	ProvideAlarmProcessor,
	ProvideHRMPersonnelService,
	ProvideHRMVacationService,
	ProvideHRMDashboardService,
	ProvideHRMTimesheetService,
	ProvideHRMSalaryService,
	ProvideHRMRecruitingService,
	ProvideHRMTrainingService,
	ProvideHRMDocumentService,
	ProvideHRMAccessService,
	ProvideHRMOrgStructureService,
	ProvideHRMCompetencyService,
	ProvideHRMPerformanceService,
	ProvideHRMAnalyticsService,
	ProvideReservoirHourlyService,
)

// ProvideTokenService creates JWT token service
func ProvideTokenService(jwtCfg config.JwtConfig) (*token.Token, error) {
	return token.New(jwtCfg.Secret, jwtCfg.AccessTimeout, jwtCfg.RefreshTimeout)
}

// ProvideASCUEFetcher creates ASCUE fetcher (returns nil if config is nil)
func ProvideASCUEFetcher(cfg *config.ASCUEConfig, log *slog.Logger) *ascue.Fetcher {
	if cfg == nil {
		return nil
	}
	return ascue.NewFetcher(cfg, log)
}

// ProvideMetricsBlender creates MetricsBlender that wraps ASCUEFetcher with ASUTP enrichment
func ProvideMetricsBlender(fetcher *ascue.Fetcher, redisRepo *redis.Repo, log *slog.Logger) *metrics.MetricsBlender {
	if fetcher == nil {
		return nil
	}
	return metrics.NewMetricsBlender(fetcher, redisRepo, log)
}

// ProvideReservoirFetcher creates reservoir fetcher (returns nil if config is nil)
func ProvideReservoirFetcher(cfg *config.ReservoirConfig, log *slog.Logger) *reservoir.Fetcher {
	if cfg == nil {
		return nil
	}

	var reservoirOrgIDs []int64
	for _, source := range cfg.Sources {
		reservoirOrgIDs = append(reservoirOrgIDs, source.OrganizationID)
	}

	return reservoir.NewFetcher(cfg, log, reservoirOrgIDs)
}

// ProvideHTTPClient creates a shared HTTP client
// This prevents creating multiple clients in router.go
func ProvideHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
	}
}

// ProvideAlarmProcessor creates the alarm processor for automatic shutdown creation
func ProvideAlarmProcessor(pgRepo *repo.Repo, redisRepo *redis.Repo, log *slog.Logger) *alarm.Processor {
	return alarm.NewProcessor(pgRepo, redisRepo, log)
}

// ProvideHRMPersonnelService creates the HRM personnel service
func ProvideHRMPersonnelService(pgRepo *repo.Repo, log *slog.Logger) *hrmpersonnel.Service {
	return hrmpersonnel.NewService(pgRepo, log)
}

// ProvideHRMVacationService creates the HRM vacation service
func ProvideHRMVacationService(pgRepo *repo.Repo, log *slog.Logger) *hrmvacation.Service {
	return hrmvacation.NewService(pgRepo, log)
}

// ProvideHRMDashboardService creates the HRM dashboard service
func ProvideHRMDashboardService(pgRepo *repo.Repo, log *slog.Logger) *hrmdashboard.Service {
	return hrmdashboard.NewService(pgRepo, log)
}

// ProvideHRMTimesheetService creates the HRM timesheet service
func ProvideHRMTimesheetService(pgRepo *repo.Repo, log *slog.Logger) *hrmtimesheet.Service {
	return hrmtimesheet.NewService(pgRepo, log)
}

// ProvideHRMSalaryService creates the HRM salary service
func ProvideHRMSalaryService(pgRepo *repo.Repo, log *slog.Logger) *hrmsalary.Service {
	return hrmsalary.NewService(pgRepo, log)
}

// ProvideHRMRecruitingService creates the HRM recruiting service
func ProvideHRMRecruitingService(pgRepo *repo.Repo, log *slog.Logger) *hrmrecruiting.Service {
	return hrmrecruiting.NewService(pgRepo, log)
}

// ProvideHRMTrainingService creates the HRM training service
func ProvideHRMTrainingService(pgRepo *repo.Repo, log *slog.Logger) *hrmtraining.Service {
	return hrmtraining.NewService(pgRepo, log)
}

// ProvideHRMDocumentService creates the HRM document service
func ProvideHRMDocumentService(pgRepo *repo.Repo, log *slog.Logger) *hrmdocument.Service {
	return hrmdocument.NewService(pgRepo, log)
}

// ProvideHRMAccessService creates the HRM access control service
func ProvideHRMAccessService(pgRepo *repo.Repo, log *slog.Logger) *hrmaccess.Service {
	return hrmaccess.NewService(pgRepo, log)
}

// ProvideHRMOrgStructureService creates the HRM org structure service
func ProvideHRMOrgStructureService(pgRepo *repo.Repo, log *slog.Logger) *hrmorgstructure.Service {
	return hrmorgstructure.NewService(pgRepo, log)
}

// ProvideHRMCompetencyService creates the HRM competency assessment service
func ProvideHRMCompetencyService(pgRepo *repo.Repo, log *slog.Logger) *hrmcompetency.Service {
	return hrmcompetency.NewService(pgRepo, log)
}

// ProvideHRMPerformanceService creates the HRM performance management service
func ProvideHRMPerformanceService(pgRepo *repo.Repo, log *slog.Logger) *hrmperformance.Service {
	return hrmperformance.NewService(pgRepo, log)
}

// ProvideHRMAnalyticsService creates the HRM analytics service
func ProvideHRMAnalyticsService(pgRepo *repo.Repo, log *slog.Logger) *hrmanalytics.Service {
	return hrmanalytics.NewService(pgRepo, log)
}

// ProvideReservoirHourlyService creates the reservoir-hourly report service
func ProvideReservoirHourlyService(fetcher *reservoir.Fetcher, pgRepo *repo.Repo, log *slog.Logger) *reservoirhourly.Service {
	if fetcher == nil {
		return nil
	}
	return reservoirhourly.NewService(fetcher, pgRepo, log)
}
