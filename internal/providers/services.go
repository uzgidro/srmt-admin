package providers

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	admincontact "srmt-admin/internal/lib/service/admin/contact"
	admindepartment "srmt-admin/internal/lib/service/admin/department"
	adminorganization "srmt-admin/internal/lib/service/admin/organization"
	adminposition "srmt-admin/internal/lib/service/admin/position"
	adminrole "srmt-admin/internal/lib/service/admin/role"
	adminuser "srmt-admin/internal/lib/service/admin/user"
	"srmt-admin/internal/lib/service/alarm"
	"srmt-admin/internal/lib/service/ascue"
	assistantevent "srmt-admin/internal/lib/service/assistant/event"
	assistantfastcall "srmt-admin/internal/lib/service/assistant/fast_call"
	chancellerydecree "srmt-admin/internal/lib/service/chancellery/decree"
	chancellerydocstatus "srmt-admin/internal/lib/service/chancellery/document_status"
	chancelleryinstruction "srmt-admin/internal/lib/service/chancellery/instruction"
	chancellerylegal "srmt-admin/internal/lib/service/chancellery/legal_document"
	chancelleryletter "srmt-admin/internal/lib/service/chancellery/letter"
	chancelleryreport "srmt-admin/internal/lib/service/chancellery/report"
	chancellerysignature "srmt-admin/internal/lib/service/chancellery/signature"
	dashboardsvc "srmt-admin/internal/lib/service/dashboard"
	dashboardcalendar "srmt-admin/internal/lib/service/dashboard/calendar"
	dataanalytics "srmt-admin/internal/lib/service/data/analytics"
	filesvc "srmt-admin/internal/lib/service/file"
	gessvc "srmt-admin/internal/lib/service/ges"
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
	ingestionsensor "srmt-admin/internal/lib/service/ingestion/sensor"
	ingestiontelemetry "srmt-admin/internal/lib/service/ingestion/telemetry"
	investactiveproject "srmt-admin/internal/lib/service/investment/active_project"
	investmentinvestment "srmt-admin/internal/lib/service/investment/investment"
	"srmt-admin/internal/lib/service/metrics"
	opsdischarge "srmt-admin/internal/lib/service/operations/discharge"
	opsincident "srmt-admin/internal/lib/service/operations/incident"
	opspastevents "srmt-admin/internal/lib/service/operations/past_events"
	opsshutdown "srmt-admin/internal/lib/service/operations/shutdown"
	opsvisit "srmt-admin/internal/lib/service/operations/visit"
	receptionsvc "srmt-admin/internal/lib/service/reception"
	"srmt-admin/internal/lib/service/reservoir"
	scdata "srmt-admin/internal/lib/service/sc/data"
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

	// Admin services
	ProvideAdminDepartmentService,
	ProvideAdminRoleService,
	ProvideAdminPositionService,
	ProvideAdminOrganizationService,
	ProvideAdminContactService,
	ProvideAdminUserService,

	// Assistant services
	ProvideAssistantFastCallService,
	ProvideAssistantEventService,

	// Reception service
	ProvideReceptionService,

	// Investment services
	ProvideInvestActiveProjectService,
	ProvideInvestmentInvestmentService,

	// Operations services
	ProvideOperationsPastEventsService,
	ProvideOperationsIncidentService,
	ProvideOperationsShutdownService,
	ProvideOperationsVisitService,
	ProvideOperationsDischargeService,

	// GES service
	ProvideGESService,

	// Dashboard services
	ProvideDashboardService,
	ProvideDashboardCalendarService,

	// SC data service
	ProvideSCDataService,

	// Chancellery services
	ProvideChancelleryDocStatusService,
	ProvideChancelleryDecreeService,
	ProvideChancelleryReportService,
	ProvideChancelleryLetterService,
	ProvideChancelleryInstructionService,
	ProvideChancelleryLegalDocumentService,
	ProvideChancellerySignatureService,

	// HRM services
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

	// File service
	ProvideFileService,

	// Data analytics service
	ProvideDataAnalyticsService,

	// Ingestion services
	ProvideIngestionSensorService,
	ProvideIngestionTelemetryService,
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

// --- Admin services ---

func ProvideAdminDepartmentService(pgRepo *repo.Repo, log *slog.Logger) *admindepartment.Service {
	return admindepartment.NewService(pgRepo, log)
}

func ProvideAdminRoleService(pgRepo *repo.Repo, log *slog.Logger) *adminrole.Service {
	return adminrole.NewService(pgRepo, log)
}

func ProvideAdminPositionService(pgRepo *repo.Repo, log *slog.Logger) *adminposition.Service {
	return adminposition.NewService(pgRepo, log)
}

func ProvideAdminOrganizationService(pgRepo *repo.Repo, log *slog.Logger) *adminorganization.Service {
	return adminorganization.NewService(pgRepo, log)
}

// --- Assistant services ---

func ProvideAssistantFastCallService(pgRepo *repo.Repo, log *slog.Logger) *assistantfastcall.Service {
	return assistantfastcall.NewService(pgRepo, log)
}

// --- Reception service ---

func ProvideReceptionService(pgRepo *repo.Repo, log *slog.Logger) *receptionsvc.Service {
	return receptionsvc.NewService(pgRepo, log)
}

// --- Investment services ---

func ProvideInvestActiveProjectService(pgRepo *repo.Repo, log *slog.Logger) *investactiveproject.Service {
	return investactiveproject.NewService(pgRepo, log)
}

// --- Admin services (file-bearing) ---

func ProvideAdminContactService(pgRepo *repo.Repo, log *slog.Logger) *admincontact.Service {
	return admincontact.NewService(pgRepo, log)
}

func ProvideAdminUserService(pgRepo *repo.Repo, log *slog.Logger) *adminuser.Service {
	return adminuser.NewService(pgRepo, log)
}

// --- Assistant services (file-bearing) ---

func ProvideAssistantEventService(pgRepo *repo.Repo, log *slog.Logger) *assistantevent.Service {
	return assistantevent.NewService(pgRepo, log)
}

// --- Investment services (file-bearing) ---

func ProvideInvestmentInvestmentService(pgRepo *repo.Repo, log *slog.Logger) *investmentinvestment.Service {
	return investmentinvestment.NewService(pgRepo, log)
}

// --- Operations services ---

func ProvideOperationsIncidentService(pgRepo *repo.Repo, log *slog.Logger) *opsincident.Service {
	return opsincident.NewService(pgRepo, log)
}

func ProvideOperationsShutdownService(pgRepo *repo.Repo, log *slog.Logger) *opsshutdown.Service {
	return opsshutdown.NewService(pgRepo, log)
}

func ProvideOperationsVisitService(pgRepo *repo.Repo, log *slog.Logger) *opsvisit.Service {
	return opsvisit.NewService(pgRepo, log)
}

func ProvideOperationsDischargeService(pgRepo *repo.Repo, log *slog.Logger) *opsdischarge.Service {
	return opsdischarge.NewService(pgRepo, log)
}

// --- Chancellery services ---

func ProvideChancelleryDocStatusService(pgRepo *repo.Repo, log *slog.Logger) *chancellerydocstatus.Service {
	return chancellerydocstatus.NewService(pgRepo, log)
}

func ProvideChancelleryDecreeService(pgRepo *repo.Repo, log *slog.Logger) *chancellerydecree.Service {
	return chancellerydecree.NewService(pgRepo, log)
}

func ProvideChancelleryReportService(pgRepo *repo.Repo, log *slog.Logger) *chancelleryreport.Service {
	return chancelleryreport.NewService(pgRepo, log)
}

func ProvideChancelleryLetterService(pgRepo *repo.Repo, log *slog.Logger) *chancelleryletter.Service {
	return chancelleryletter.NewService(pgRepo, log)
}

func ProvideChancelleryInstructionService(pgRepo *repo.Repo, log *slog.Logger) *chancelleryinstruction.Service {
	return chancelleryinstruction.NewService(pgRepo, log)
}

func ProvideChancelleryLegalDocumentService(pgRepo *repo.Repo, log *slog.Logger) *chancellerylegal.Service {
	return chancellerylegal.NewService(pgRepo, log)
}

func ProvideChancellerySignatureService(pgRepo *repo.Repo, log *slog.Logger) *chancellerysignature.Service {
	return chancellerysignature.NewService(pgRepo, log)
}

// --- Phase 5: Read-only aggregation services ---

func ProvideOperationsPastEventsService(pgRepo *repo.Repo, log *slog.Logger) *opspastevents.Service {
	return opspastevents.NewService(pgRepo, log)
}

func ProvideGESService(pgRepo *repo.Repo, log *slog.Logger) *gessvc.Service {
	return gessvc.NewService(pgRepo, log)
}

func ProvideDashboardService(pgRepo *repo.Repo, log *slog.Logger) *dashboardsvc.Service {
	return dashboardsvc.NewService(pgRepo, log)
}

func ProvideDashboardCalendarService(pgRepo *repo.Repo, log *slog.Logger) *dashboardcalendar.Service {
	return dashboardcalendar.NewService(pgRepo, log)
}

func ProvideSCDataService(pgRepo *repo.Repo, log *slog.Logger) *scdata.Service {
	return scdata.NewService(pgRepo, log)
}

func ProvideFileService(pgRepo *repo.Repo, log *slog.Logger) *filesvc.Service {
	return filesvc.NewService(pgRepo, log)
}

func ProvideDataAnalyticsService(pgRepo *repo.Repo, log *slog.Logger) *dataanalytics.Service {
	return dataanalytics.NewService(pgRepo, log)
}

func ProvideIngestionSensorService(pgRepo *repo.Repo, log *slog.Logger) *ingestionsensor.Service {
	return ingestionsensor.NewService(pgRepo, log)
}

func ProvideIngestionTelemetryService(redisRepo *redis.Repo, log *slog.Logger) *ingestiontelemetry.Service {
	return ingestiontelemetry.NewService(redisRepo, log)
}
