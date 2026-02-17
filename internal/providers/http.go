package providers

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/middleware/cors"
	"srmt-admin/internal/http-server/middleware/logger"
	"srmt-admin/internal/http-server/router"
	admincontact "srmt-admin/internal/lib/service/admin/contact"
	admindepartment "srmt-admin/internal/lib/service/admin/department"
	adminorganization "srmt-admin/internal/lib/service/admin/organization"
	adminposition "srmt-admin/internal/lib/service/admin/position"
	adminrole "srmt-admin/internal/lib/service/admin/role"
	adminuser "srmt-admin/internal/lib/service/admin/user"
	"srmt-admin/internal/lib/service/alarm"
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
	"srmt-admin/internal/storage/minio"
	mngRepo "srmt-admin/internal/storage/mongo"
	redisRepo "srmt-admin/internal/storage/redis"
	pgRepo "srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/wire"
)

// HTTPProviderSet provides HTTP server dependencies
var HTTPProviderSet = wire.NewSet(
	ProvideRouter,
	ProvideHTTPServer,
	ProvideAppContainer,
)

// AppContainer holds all application dependencies
type AppContainer struct {
	Router           *chi.Mux
	Server           *http.Server
	Logger           *slog.Logger
	Config           *config.Config
	PgRepo           *pgRepo.Repo
	MongoRepo        *mngRepo.Repo
	MinioRepo        *minio.Repo
	RedisRepo        *redisRepo.Repo
	Token            *token.Token
	Location         *time.Location
	MetricsBlender   *metrics.MetricsBlender
	ReservoirFetcher *reservoir.Fetcher
	HTTPClient       *http.Client
	AlarmProcessor   *alarm.Processor

	// Admin services
	AdminDepartmentService   *admindepartment.Service
	AdminRoleService         *adminrole.Service
	AdminPositionService     *adminposition.Service
	AdminOrganizationService *adminorganization.Service
	AdminContactService      *admincontact.Service
	AdminUserService         *adminuser.Service

	// Assistant services
	AssistantFastCallService *assistantfastcall.Service
	AssistantEventService    *assistantevent.Service

	// Reception service
	ReceptionService *receptionsvc.Service

	// Investment services
	InvestActiveProjectService  *investactiveproject.Service
	InvestmentInvestmentService *investmentinvestment.Service

	// Operations services
	OperationsPastEventsService *opspastevents.Service
	OperationsIncidentService   *opsincident.Service
	OperationsShutdownService   *opsshutdown.Service
	OperationsVisitService      *opsvisit.Service
	OperationsDischargeService  *opsdischarge.Service

	// GES service
	GESService *gessvc.Service

	// Dashboard services
	DashboardService         *dashboardsvc.Service
	DashboardCalendarService *dashboardcalendar.Service

	// SC data service
	SCDataService *scdata.Service

	// File service
	FileService *filesvc.Service

	// Data analytics service
	DataAnalyticsService *dataanalytics.Service

	// Ingestion services
	IngestionSensorService    *ingestionsensor.Service
	IngestionTelemetryService *ingestiontelemetry.Service

	// Chancellery services
	ChancelleryDocStatusService   *chancellerydocstatus.Service
	ChancelleryDecreeService      *chancellerydecree.Service
	ChancelleryReportService      *chancelleryreport.Service
	ChancelleryLetterService      *chancelleryletter.Service
	ChancelleryInstructionService *chancelleryinstruction.Service
	ChancelleryLegalDocService    *chancellerylegal.Service
	ChancellerySignatureService   *chancellerysignature.Service

	// HRM services
	HRMPersonnelService    *hrmpersonnel.Service
	HRMVacationService     *hrmvacation.Service
	HRMDashboardService    *hrmdashboard.Service
	HRMTimesheetService    *hrmtimesheet.Service
	HRMSalaryService       *hrmsalary.Service
	HRMRecruitingService   *hrmrecruiting.Service
	HRMTrainingService     *hrmtraining.Service
	HRMDocumentService     *hrmdocument.Service
	HRMAccessService       *hrmaccess.Service
	HRMOrgStructureService *hrmorgstructure.Service
	HRMCompetencyService   *hrmcompetency.Service
	HRMPerformanceService  *hrmperformance.Service
	HRMAnalyticsService    *hrmanalytics.Service
}

// ProvideAppContainer creates the application container
func ProvideAppContainer(
	r *chi.Mux,
	srv *http.Server,
	log *slog.Logger,
	cfg *config.Config,
	pg *pgRepo.Repo,
	mng *mngRepo.Repo,
	minioRepo *minio.Repo,
	redis *redisRepo.Repo,
	tkn *token.Token,
	loc *time.Location,
	metricsBlender *metrics.MetricsBlender,
	reservoirFetcher *reservoir.Fetcher,
	httpClient *http.Client,
	alarmProcessor *alarm.Processor,
	adminDeptSvc *admindepartment.Service,
	adminRoleSvc *adminrole.Service,
	adminPosSvc *adminposition.Service,
	adminOrgSvc *adminorganization.Service,
	adminContactSvc *admincontact.Service,
	adminUserSvc *adminuser.Service,
	fastCallSvc *assistantfastcall.Service,
	eventSvc *assistantevent.Service,
	receptionSvc *receptionsvc.Service,
	investActiveProjectSvc *investactiveproject.Service,
	investmentSvc *investmentinvestment.Service,
	incidentSvc *opsincident.Service,
	shutdownSvc *opsshutdown.Service,
	visitSvc *opsvisit.Service,
	pastEventsSvc *opspastevents.Service,
	dischargeSvc *opsdischarge.Service,
	gesSvc *gessvc.Service,
	dashboardSvc *dashboardsvc.Service,
	dashboardCalendarSvc *dashboardcalendar.Service,
	scDataSvc *scdata.Service,
	fileSvc *filesvc.Service,
	dataAnalyticsSvc *dataanalytics.Service,
	ingestionSensorSvc *ingestionsensor.Service,
	ingestionTelemetrySvc *ingestiontelemetry.Service,
	docStatusSvc *chancellerydocstatus.Service,
	decreeSvc *chancellerydecree.Service,
	reportSvc *chancelleryreport.Service,
	letterSvc *chancelleryletter.Service,
	instructionSvc *chancelleryinstruction.Service,
	legalDocSvc *chancellerylegal.Service,
	signatureSvc *chancellerysignature.Service,
	hrmPersonnelSvc *hrmpersonnel.Service,
	hrmVacationSvc *hrmvacation.Service,
	hrmDashboardSvc *hrmdashboard.Service,
	hrmTimesheetSvc *hrmtimesheet.Service,
	hrmSalarySvc *hrmsalary.Service,
	hrmRecruitingSvc *hrmrecruiting.Service,
	hrmTrainingSvc *hrmtraining.Service,
	hrmDocumentSvc *hrmdocument.Service,
	hrmAccessSvc *hrmaccess.Service,
	hrmOrgStructureSvc *hrmorgstructure.Service,
	hrmCompetencySvc *hrmcompetency.Service,
	hrmPerformanceSvc *hrmperformance.Service,
	hrmAnalyticsSvc *hrmanalytics.Service,
) *AppContainer {
	return &AppContainer{
		Router:                        r,
		Server:                        srv,
		Logger:                        log,
		Config:                        cfg,
		PgRepo:                        pg,
		MongoRepo:                     mng,
		MinioRepo:                     minioRepo,
		RedisRepo:                     redis,
		Token:                         tkn,
		Location:                      loc,
		MetricsBlender:                metricsBlender,
		ReservoirFetcher:              reservoirFetcher,
		HTTPClient:                    httpClient,
		AlarmProcessor:                alarmProcessor,
		AdminDepartmentService:        adminDeptSvc,
		AdminRoleService:              adminRoleSvc,
		AdminPositionService:          adminPosSvc,
		AdminOrganizationService:      adminOrgSvc,
		AdminContactService:           adminContactSvc,
		AdminUserService:              adminUserSvc,
		AssistantFastCallService:      fastCallSvc,
		AssistantEventService:         eventSvc,
		ReceptionService:              receptionSvc,
		InvestActiveProjectService:    investActiveProjectSvc,
		InvestmentInvestmentService:   investmentSvc,
		OperationsIncidentService:     incidentSvc,
		OperationsShutdownService:     shutdownSvc,
		OperationsVisitService:        visitSvc,
		OperationsPastEventsService:   pastEventsSvc,
		OperationsDischargeService:    dischargeSvc,
		GESService:                    gesSvc,
		DashboardService:              dashboardSvc,
		DashboardCalendarService:      dashboardCalendarSvc,
		SCDataService:                 scDataSvc,
		FileService:                   fileSvc,
		DataAnalyticsService:          dataAnalyticsSvc,
		IngestionSensorService:        ingestionSensorSvc,
		IngestionTelemetryService:     ingestionTelemetrySvc,
		ChancelleryDocStatusService:   docStatusSvc,
		ChancelleryDecreeService:      decreeSvc,
		ChancelleryReportService:      reportSvc,
		ChancelleryLetterService:      letterSvc,
		ChancelleryInstructionService: instructionSvc,
		ChancelleryLegalDocService:    legalDocSvc,
		ChancellerySignatureService:   signatureSvc,
		HRMPersonnelService:           hrmPersonnelSvc,
		HRMVacationService:            hrmVacationSvc,
		HRMDashboardService:           hrmDashboardSvc,
		HRMTimesheetService:           hrmTimesheetSvc,
		HRMSalaryService:              hrmSalarySvc,
		HRMRecruitingService:          hrmRecruitingSvc,
		HRMTrainingService:            hrmTrainingSvc,
		HRMDocumentService:            hrmDocumentSvc,
		HRMAccessService:              hrmAccessSvc,
		HRMOrgStructureService:        hrmOrgStructureSvc,
		HRMCompetencyService:          hrmCompetencySvc,
		HRMPerformanceService:         hrmPerformanceSvc,
		HRMAnalyticsService:           hrmAnalyticsSvc,
	}
}

// ProvideRouter creates and configures the chi router
func ProvideRouter(
	log *slog.Logger,
	cfg *config.Config,
	tkn *token.Token,
	pg *pgRepo.Repo,
	mng *mngRepo.Repo,
	minioRepo *minio.Repo,
	redis *redisRepo.Repo,
	loc *time.Location,
	metricsBlender *metrics.MetricsBlender,
	reservoirFetcher *reservoir.Fetcher,
	httpClient *http.Client,
	alarmProcessor *alarm.Processor,
	adminDeptSvc *admindepartment.Service,
	adminRoleSvc *adminrole.Service,
	adminPosSvc *adminposition.Service,
	adminOrgSvc *adminorganization.Service,
	adminContactSvc *admincontact.Service,
	adminUserSvc *adminuser.Service,
	fastCallSvc *assistantfastcall.Service,
	eventSvc *assistantevent.Service,
	receptionSvc *receptionsvc.Service,
	investActiveProjectSvc *investactiveproject.Service,
	investmentSvc *investmentinvestment.Service,
	incidentSvc *opsincident.Service,
	shutdownSvc *opsshutdown.Service,
	visitSvc *opsvisit.Service,
	dischargeSvc *opsdischarge.Service,
	docStatusSvc *chancellerydocstatus.Service,
	decreeSvc *chancellerydecree.Service,
	reportSvc *chancelleryreport.Service,
	letterSvc *chancelleryletter.Service,
	instructionSvc *chancelleryinstruction.Service,
	legalDocSvc *chancellerylegal.Service,
	signatureSvc *chancellerysignature.Service,
	pastEventsSvc *opspastevents.Service,
	gesSvc *gessvc.Service,
	dashboardSvc *dashboardsvc.Service,
	dashboardCalendarSvc *dashboardcalendar.Service,
	scDataSvc *scdata.Service,
	fileSvc *filesvc.Service,
	dataAnalyticsSvc *dataanalytics.Service,
	ingestionSensorSvc *ingestionsensor.Service,
	ingestionTelemetrySvc *ingestiontelemetry.Service,
	hrmPersonnelSvc *hrmpersonnel.Service,
	hrmVacationSvc *hrmvacation.Service,
	hrmDashboardSvc *hrmdashboard.Service,
	hrmTimesheetSvc *hrmtimesheet.Service,
	hrmSalarySvc *hrmsalary.Service,
	hrmRecruitingSvc *hrmrecruiting.Service,
	hrmTrainingSvc *hrmtraining.Service,
	hrmDocumentSvc *hrmdocument.Service,
	hrmAccessSvc *hrmaccess.Service,
	hrmOrgStructureSvc *hrmorgstructure.Service,
	hrmCompetencySvc *hrmcompetency.Service,
	hrmPerformanceSvc *hrmperformance.Service,
	hrmAnalyticsSvc *hrmanalytics.Service,
) *chi.Mux {
	r := chi.NewRouter()

	// Middleware stack (moved from main.go)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.New(log))
	r.Use(middleware.Recoverer)
	r.Use(cors.New(cfg.AllowedOrigins))

	// Setup routes with new container pattern
	deps := &router.AppDependencies{
		Log:                           log,
		Token:                         tkn,
		PgRepo:                        pg,
		MongoRepo:                     mng,
		MinioRepo:                     minioRepo,
		RedisRepo:                     redis,
		Config:                        *cfg,
		Location:                      loc,
		MetricsBlender:                metricsBlender,
		ReservoirFetcher:              reservoirFetcher,
		HTTPClient:                    httpClient,
		ExcelTemplatePath:             cfg.TemplatePath + "/res-summary.xlsx",
		DischargeExcelTemplatePath:    cfg.TemplatePath + "/discharge.xlsx",
		SCExcelTemplatePath:           cfg.TemplatePath + "/sc.xlsx",
		AlarmProcessor:                alarmProcessor,
		AdminDepartmentService:        adminDeptSvc,
		AdminRoleService:              adminRoleSvc,
		AdminPositionService:          adminPosSvc,
		AdminOrganizationService:      adminOrgSvc,
		AdminContactService:           adminContactSvc,
		AdminUserService:              adminUserSvc,
		AssistantFastCallService:      fastCallSvc,
		AssistantEventService:         eventSvc,
		ReceptionService:              receptionSvc,
		InvestActiveProjectService:    investActiveProjectSvc,
		InvestmentInvestmentService:   investmentSvc,
		OperationsIncidentService:     incidentSvc,
		OperationsShutdownService:     shutdownSvc,
		OperationsVisitService:        visitSvc,
		OperationsDischargeService:    dischargeSvc,
		ChancelleryDocStatusService:   docStatusSvc,
		ChancelleryDecreeService:      decreeSvc,
		ChancelleryReportService:      reportSvc,
		ChancelleryLetterService:      letterSvc,
		ChancelleryInstructionService: instructionSvc,
		ChancelleryLegalDocService:    legalDocSvc,
		ChancellerySignatureService:   signatureSvc,
		OperationsPastEventsService:   pastEventsSvc,
		GESService:                    gesSvc,
		DashboardService:              dashboardSvc,
		DashboardCalendarService:      dashboardCalendarSvc,
		SCDataService:                 scDataSvc,
		FileService:                   fileSvc,
		DataAnalyticsService:          dataAnalyticsSvc,
		IngestionSensorService:        ingestionSensorSvc,
		IngestionTelemetryService:     ingestionTelemetrySvc,
		HRMPersonnelService:           hrmPersonnelSvc,
		HRMVacationService:            hrmVacationSvc,
		HRMDashboardService:           hrmDashboardSvc,
		HRMTimesheetService:           hrmTimesheetSvc,
		HRMSalaryService:              hrmSalarySvc,
		HRMRecruitingService:          hrmRecruitingSvc,
		HRMTrainingService:            hrmTrainingSvc,
		HRMDocumentService:            hrmDocumentSvc,
		HRMAccessService:              hrmAccessSvc,
		HRMOrgStructureService:        hrmOrgStructureSvc,
		HRMCompetencyService:          hrmCompetencySvc,
		HRMPerformanceService:         hrmPerformanceSvc,
		HRMAnalyticsService:           hrmAnalyticsSvc,
	}

	router.SetupRoutes(r, deps)

	return r
}

// ProvideHTTPServer creates the HTTP server
func ProvideHTTPServer(r *chi.Mux, cfg *config.Config) *http.Server {
	return &http.Server{
		Addr:         cfg.HttpServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HttpServer.Timeout,
		WriteTimeout: cfg.HttpServer.Timeout,
		IdleTimeout:  cfg.HttpServer.IdleTimeout,
	}
}
