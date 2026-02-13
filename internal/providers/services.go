package providers

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/lib/service/alarm"
	"srmt-admin/internal/lib/service/ascue"
	hrmdashboard "srmt-admin/internal/lib/service/hrm/dashboard"
	hrmpersonnel "srmt-admin/internal/lib/service/hrm/personnel"
	hrmsalary "srmt-admin/internal/lib/service/hrm/salary"
	hrmtimesheet "srmt-admin/internal/lib/service/hrm/timesheet"
	hrmvacation "srmt-admin/internal/lib/service/hrm/vacation"
	"srmt-admin/internal/lib/service/metrics"
	"srmt-admin/internal/lib/service/reservoir"
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
