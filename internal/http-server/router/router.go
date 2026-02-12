package router

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	asutpTelemetry "srmt-admin/internal/http-server/handlers/asutp/telemetry"
	"srmt-admin/internal/http-server/handlers/auth/me"
	"srmt-admin/internal/http-server/handlers/auth/refresh"
	signIn "srmt-admin/internal/http-server/handlers/auth/sign-in"
	signOut "srmt-admin/internal/http-server/handlers/auth/sign-out"
	"srmt-admin/internal/http-server/handlers/calendar"
	contactAdd "srmt-admin/internal/http-server/handlers/contacts/add"
	contactDelete "srmt-admin/internal/http-server/handlers/contacts/delete"
	contactEdit "srmt-admin/internal/http-server/handlers/contacts/edit"
	contactGetAll "srmt-admin/internal/http-server/handlers/contacts/get-all"
	contactGetById "srmt-admin/internal/http-server/handlers/contacts/get-by-id"
	"srmt-admin/internal/http-server/handlers/currency"
	orgGetCascades "srmt-admin/internal/http-server/handlers/dashboard/get-cascades"
	dashboardGetReservoir "srmt-admin/internal/http-server/handlers/dashboard/get-reservoir"
	dashboardGetReservoirHourly "srmt-admin/internal/http-server/handlers/dashboard/get-reservoir-hourly"
	"srmt-admin/internal/http-server/handlers/dashboard/production"
	productionstats "srmt-admin/internal/http-server/handlers/dashboard/production-stats"
	"srmt-admin/internal/http-server/handlers/data/analytics"
	dataSet "srmt-admin/internal/http-server/handlers/data/set"
	"srmt-admin/internal/http-server/handlers/decrees"
	departmentAdd "srmt-admin/internal/http-server/handlers/department/add"
	departmentDelete "srmt-admin/internal/http-server/handlers/department/delete"
	departmentEdit "srmt-admin/internal/http-server/handlers/department/edit"
	departmentGetAll "srmt-admin/internal/http-server/handlers/department/get-all"
	departmentGetById "srmt-admin/internal/http-server/handlers/department/get-by-id"
	dischargeAdd "srmt-admin/internal/http-server/handlers/discharge/add"
	dischargeDelete "srmt-admin/internal/http-server/handlers/discharge/delete"
	dischargePatch "srmt-admin/internal/http-server/handlers/discharge/edit"
	dischargeExport "srmt-admin/internal/http-server/handlers/discharge/export"
	dischargeGet "srmt-admin/internal/http-server/handlers/discharge/get"
	dischargeGetCurrent "srmt-admin/internal/http-server/handlers/discharge/get-current"
	dischargeGetFlat "srmt-admin/internal/http-server/handlers/discharge/get-flat"
	docstatuses "srmt-admin/internal/http-server/handlers/document-statuses"
	eventAdd "srmt-admin/internal/http-server/handlers/events/add"
	eventDelete "srmt-admin/internal/http-server/handlers/events/delete"
	eventEdit "srmt-admin/internal/http-server/handlers/events/edit"
	eventGetAll "srmt-admin/internal/http-server/handlers/events/get-all"
	eventGetById "srmt-admin/internal/http-server/handlers/events/get-by-id"
	eventGetShort "srmt-admin/internal/http-server/handlers/events/get-short"
	eventGetStatuses "srmt-admin/internal/http-server/handlers/events/get-statuses"
	eventGetTypes "srmt-admin/internal/http-server/handlers/events/get-types"
	fastCallAdd "srmt-admin/internal/http-server/handlers/fast-call/add"
	fastCallDelete "srmt-admin/internal/http-server/handlers/fast-call/delete"
	fastCallEdit "srmt-admin/internal/http-server/handlers/fast-call/edit"
	fastCallGetAll "srmt-admin/internal/http-server/handlers/fast-call/get-all"
	fastCallGetById "srmt-admin/internal/http-server/handlers/fast-call/get-by-id"
	catAdd "srmt-admin/internal/http-server/handlers/file/category/add"
	catGet "srmt-admin/internal/http-server/handlers/file/category/list"
	fileDelete "srmt-admin/internal/http-server/handlers/file/delete"
	"srmt-admin/internal/http-server/handlers/file/download"
	getbycategory "srmt-admin/internal/http-server/handlers/file/get-by-category"
	"srmt-admin/internal/http-server/handlers/file/latest"
	"srmt-admin/internal/http-server/handlers/file/upload"
	gesAskue "srmt-admin/internal/http-server/handlers/ges/askue"
	gesContacts "srmt-admin/internal/http-server/handlers/ges/contacts"
	gesDepartments "srmt-admin/internal/http-server/handlers/ges/departments"
	gesDischarges "srmt-admin/internal/http-server/handlers/ges/discharges"
	gesGet "srmt-admin/internal/http-server/handlers/ges/get"
	gesIncidents "srmt-admin/internal/http-server/handlers/ges/incidents"
	gesShutdowns "srmt-admin/internal/http-server/handlers/ges/shutdowns"
	gesVisits "srmt-admin/internal/http-server/handlers/ges/visits"
	hrmDashboardHandler "srmt-admin/internal/http-server/handlers/hrm/dashboard"
	hrmPersonnelHandler "srmt-admin/internal/http-server/handlers/hrm/personnel"
	hrmVacationHandler "srmt-admin/internal/http-server/handlers/hrm/vacation"
	incidentsHandler "srmt-admin/internal/http-server/handlers/incidents-handler"
	setIndicator "srmt-admin/internal/http-server/handlers/indicators/set"
	"srmt-admin/internal/http-server/handlers/instructions"
	investActiveProjects "srmt-admin/internal/http-server/handlers/invest-active-projects"
	"srmt-admin/internal/http-server/handlers/investments"
	legaldocuments "srmt-admin/internal/http-server/handlers/legal-documents"
	"srmt-admin/internal/http-server/handlers/letters"
	levelVolumeGet "srmt-admin/internal/http-server/handlers/level-volume/get"
	lexparser "srmt-admin/internal/http-server/handlers/lex-parser"
	myCompetencies "srmt-admin/internal/http-server/handlers/my/competencies"
	myDocuments "srmt-admin/internal/http-server/handlers/my/documents"
	myLeaveBalance "srmt-admin/internal/http-server/handlers/my/leave-balance"
	myNotifications "srmt-admin/internal/http-server/handlers/my/notifications"
	myProfile "srmt-admin/internal/http-server/handlers/my/profile"
	mySalary "srmt-admin/internal/http-server/handlers/my/salary"
	myTasks "srmt-admin/internal/http-server/handlers/my/tasks"
	myTraining "srmt-admin/internal/http-server/handlers/my/training"
	myVacations "srmt-admin/internal/http-server/handlers/my/vacations"
	"srmt-admin/internal/http-server/handlers/news"
	orgTypeAdd "srmt-admin/internal/http-server/handlers/organization-types/add"
	orgTypeDelete "srmt-admin/internal/http-server/handlers/organization-types/delete"
	orgTypeGet "srmt-admin/internal/http-server/handlers/organization-types/get"
	orgAdd "srmt-admin/internal/http-server/handlers/organizations/add"
	orgDelete "srmt-admin/internal/http-server/handlers/organizations/delete"
	orgPatch "srmt-admin/internal/http-server/handlers/organizations/edit"
	orgGet "srmt-admin/internal/http-server/handlers/organizations/get"
	orgGetFlat "srmt-admin/internal/http-server/handlers/organizations/get-flat"
	pastEventsHandler "srmt-admin/internal/http-server/handlers/past-events-handler"
	positionsAdd "srmt-admin/internal/http-server/handlers/positions/add"
	positionsDelete "srmt-admin/internal/http-server/handlers/positions/delete"
	positionsGet "srmt-admin/internal/http-server/handlers/positions/get"
	positionsPatch "srmt-admin/internal/http-server/handlers/positions/patch"
	receptionAdd "srmt-admin/internal/http-server/handlers/reception/add"
	receptionDelete "srmt-admin/internal/http-server/handlers/reception/delete"
	receptionEdit "srmt-admin/internal/http-server/handlers/reception/edit"
	receptionGetAll "srmt-admin/internal/http-server/handlers/reception/get-all"
	receptionGetById "srmt-admin/internal/http-server/handlers/reception/get-by-id"
	"srmt-admin/internal/http-server/handlers/reports"
	reservoirdevicesummary "srmt-admin/internal/http-server/handlers/reservoir-device-summary"
	reservoirsummary "srmt-admin/internal/http-server/handlers/reservoir-summary"
	resAdd "srmt-admin/internal/http-server/handlers/reservoirs/add"
	roleAdd "srmt-admin/internal/http-server/handlers/role/add"
	roleDelete "srmt-admin/internal/http-server/handlers/role/delete"
	roleEdit "srmt-admin/internal/http-server/handlers/role/edit"
	roleGet "srmt-admin/internal/http-server/handlers/role/get"
	gessummary "srmt-admin/internal/http-server/handlers/sc/callback/ges-summary"
	callbackModsnow "srmt-admin/internal/http-server/handlers/sc/callback/modsnow"
	callbackStock "srmt-admin/internal/http-server/handlers/sc/callback/stock"
	"srmt-admin/internal/http-server/handlers/sc/dc"
	scExport "srmt-admin/internal/http-server/handlers/sc/export"
	modsnowImg "srmt-admin/internal/http-server/handlers/sc/modsnow/img"
	"srmt-admin/internal/http-server/handlers/sc/modsnow/table"
	"srmt-admin/internal/http-server/handlers/sc/stock"
	"srmt-admin/internal/http-server/handlers/shutdowns"
	"srmt-admin/internal/http-server/handlers/signatures"
	snowCover "srmt-admin/internal/http-server/handlers/snow-cover"
	snowCoverGet "srmt-admin/internal/http-server/handlers/snow-cover/get"
	"srmt-admin/internal/http-server/handlers/telegram/gidro/test"
	usersAdd "srmt-admin/internal/http-server/handlers/users/add"
	assignRole "srmt-admin/internal/http-server/handlers/users/assign-role"
	usersDelete "srmt-admin/internal/http-server/handlers/users/delete"
	usersEdit "srmt-admin/internal/http-server/handlers/users/edit"
	usersGet "srmt-admin/internal/http-server/handlers/users/get"
	usersGetById "srmt-admin/internal/http-server/handlers/users/get-by-id"
	revokeRole "srmt-admin/internal/http-server/handlers/users/revoke-role"
	"srmt-admin/internal/http-server/handlers/visit"
	weatherProxy "srmt-admin/internal/http-server/handlers/weather/proxy"
	mwapikey "srmt-admin/internal/http-server/middleware/api-key"
	asutpauth "srmt-admin/internal/http-server/middleware/asutp-auth"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/service/alarm"
	dischargeExcelGen "srmt-admin/internal/lib/service/excel/discharge"
	excelgen "srmt-admin/internal/lib/service/excel/reservoir-summary"
	scExcelGen "srmt-admin/internal/lib/service/excel/sc"
	hrmdashboard "srmt-admin/internal/lib/service/hrm/dashboard"
	hrmpersonnel "srmt-admin/internal/lib/service/hrm/personnel"
	hrmvacation "srmt-admin/internal/lib/service/hrm/vacation"
	"srmt-admin/internal/lib/service/metrics"
	"srmt-admin/internal/lib/service/reservoir"
	"srmt-admin/internal/storage/minio"
	"srmt-admin/internal/storage/mongo"
	redisRepo "srmt-admin/internal/storage/redis"
	"srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
	"time"

	"github.com/go-chi/chi/v5"
)

// AppDependencies contains all dependencies needed for route setup
type AppDependencies struct {
	Log                        *slog.Logger
	Token                      *token.Token
	PgRepo                     *repo.Repo
	MongoRepo                  *mongo.Repo
	MinioRepo                  *minio.Repo
	RedisRepo                  *redisRepo.Repo
	Config                     config.Config
	Location                   *time.Location
	MetricsBlender             *metrics.MetricsBlender
	ReservoirFetcher           *reservoir.Fetcher
	HTTPClient                 *http.Client
	ExcelTemplatePath          string
	DischargeExcelTemplatePath string
	SCExcelTemplatePath        string
	AlarmProcessor             *alarm.Processor
	HRMPersonnelService        *hrmpersonnel.Service
	HRMVacationService         *hrmvacation.Service
	HRMDashboardService        *hrmdashboard.Service
}

func SetupRoutes(router *chi.Mux, deps *AppDependencies) {
	loc := deps.Location

	router.Post("/auth/sign-in", signIn.New(deps.Log, deps.PgRepo, deps.Token))
	router.Post("/auth/refresh", refresh.New(deps.Log, deps.PgRepo, deps.Token))
	router.Post("/auth/sign-out", signOut.New(deps.Log))

	router.Route("/api/v3", func(r chi.Router) {
		r.Get("/modsnow", table.Get(deps.Log, deps.MongoRepo))
		r.Get("/stock", stock.Get(deps.Log, deps.MongoRepo))
		r.Get("/modsnow/cover", modsnowImg.Get(deps.Log, deps.MinioRepo, "modsnow-cover"))
		r.Get("/modsnow/dynamics", modsnowImg.Get(deps.Log, deps.MinioRepo, "modsnow-dynamics"))

		r.Get("/analytics", analytics.New(deps.Log, deps.PgRepo))
		r.Get("/currency", currency.Get(deps.Log, deps.HTTPClient))

		r.Route("/weather", func(r chi.Router) {
			weatherCfg := deps.Config.Weather

			r.Get("/", weatherProxy.New(deps.Log, deps.HTTPClient, weatherCfg.BaseURL, weatherCfg.APIKey, "/weather"))
			r.Get("/forecast", weatherProxy.New(deps.Log, deps.HTTPClient, weatherCfg.BaseURL, weatherCfg.APIKey, "/forecast"))
		})

		r.Get("/telegram/gidro/test", test.New(deps.Log, deps.MongoRepo))
	})

	// Service endpoints
	router.Group(func(r chi.Router) {
		r.Use(mwapikey.RequireAPIKey(deps.Config.ApiKey))

		r.Post("/sc/stock", callbackStock.New(deps.Log, deps.MongoRepo))
		r.Post("/sc/modsnow", callbackModsnow.New(deps.Log, deps.MongoRepo))
		r.Post("/sc/ges/summary", gessummary.New(deps.Log, deps.PgRepo))
		r.Post("/data/{id}", dataSet.New(deps.Log, deps.PgRepo))
	})

	// Token required routes
	router.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(deps.Token))

		r.Get("/auth/me", me.New(deps.Log))

		// Personal Cabinet — any authenticated user
		r.Route("/my-profile", func(r chi.Router) {
			r.Get("/", myProfile.Get(deps.Log, deps.PgRepo))
			r.Patch("/", myProfile.Update(deps.Log, deps.PgRepo))
		})
		r.Get("/my-leave-balance", myLeaveBalance.Get(deps.Log, deps.HRMVacationService))
		r.Route("/my-vacations", func(r chi.Router) {
			r.Get("/", myVacations.GetAll(deps.Log, deps.HRMVacationService))
			r.Post("/", myVacations.Create(deps.Log, deps.HRMVacationService))
			r.Post("/{id}/cancel", myVacations.Cancel(deps.Log, deps.HRMVacationService))
		})
		r.Route("/my-notifications", func(r chi.Router) {
			r.Get("/", myNotifications.GetAll(deps.Log, deps.PgRepo))
			r.Patch("/{id}/read", myNotifications.MarkRead(deps.Log, deps.HRMDashboardService))
			r.Post("/read-all", myNotifications.MarkReadAll(deps.Log, deps.HRMDashboardService))
		})
		r.Get("/my-tasks", myTasks.GetAll(deps.Log, deps.PgRepo))
		r.Route("/my-documents", func(r chi.Router) {
			r.Get("/", myDocuments.GetAll(deps.Log, deps.HRMPersonnelService))
			r.Get("/{id}/download", myDocuments.Download(deps.Log))
		})
		r.Route("/my-salary", func(r chi.Router) {
			r.Get("/", mySalary.Get(deps.Log))
			r.Get("/payslip/{id}", mySalary.GetPayslip(deps.Log))
		})
		r.Get("/my-training", myTraining.Get(deps.Log))
		r.Get("/my-competencies", myCompetencies.Get(deps.Log))

		// News
		r.Get("/news", news.New(deps.Log, deps.HTTPClient, deps.Config.NewsRetriever.BaseURL))

		r.Get("/organization-type", orgTypeGet.New(deps.Log, deps.PgRepo))
		r.Post("/organization-type", orgTypeAdd.New(deps.Log, deps.PgRepo))
		r.Delete("/organization-type/{id}", orgTypeDelete.New(deps.Log, deps.PgRepo))

		r.Get("/department", departmentGetAll.New(deps.Log, deps.PgRepo))
		r.Get("/department/{id}", departmentGetById.New(deps.Log, deps.PgRepo))
		r.Post("/department", departmentAdd.New(deps.Log, deps.PgRepo))
		r.Patch("/department/{id}", departmentEdit.New(deps.Log, deps.PgRepo))
		r.Delete("/department/{id}", departmentDelete.New(deps.Log, deps.PgRepo))

		// Organizations
		r.Get("/organizations", orgGet.New(deps.Log, deps.PgRepo))
		r.Get("/organizations/flat", orgGetFlat.New(deps.Log, deps.PgRepo))
		r.Post("/organizations", orgAdd.New(deps.Log, deps.PgRepo))
		r.Patch("/organizations/{id}", orgPatch.New(deps.Log, deps.PgRepo))
		r.Delete("/organizations/{id}", orgDelete.New(deps.Log, deps.PgRepo))

		// Contacts
		r.Get("/contacts", contactGetAll.New(deps.Log, deps.PgRepo, deps.MinioRepo))
		r.Get("/contacts/{id}", contactGetById.New(deps.Log, deps.PgRepo, deps.MinioRepo))
		r.Post("/contacts", contactAdd.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo))
		r.Patch("/contacts/{id}", contactEdit.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo))
		r.Delete("/contacts/{id}", contactDelete.New(deps.Log, deps.PgRepo))

		// Dashboard
		r.Get("/dashboard/reservoir", dashboardGetReservoir.New(deps.Log, deps.PgRepo, deps.ReservoirFetcher))
		r.Get("/dashboard/reservoir-hourly", dashboardGetReservoirHourly.New(deps.Log, deps.ReservoirFetcher))
		r.Get("/dashboard/cascades", orgGetCascades.New(deps.Log, deps.PgRepo, deps.MetricsBlender))
		r.Get("/dashboard/production", production.New(deps.Log, deps.PgRepo))
		r.Get("/dashboard/production-stats", productionstats.New(deps.Log, deps.PgRepo))

		// Snow cover (modsnow)
		r.Get("/snow-cover", snowCoverGet.Get(deps.Log, deps.PgRepo))

		// Open routes (available to all authenticated users)
		r.Get("/shutdowns", shutdowns.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
		r.Get("/legal-documents", legaldocuments.GetAll(deps.Log, deps.PgRepo, deps.MinioRepo))
		r.Get("/legal-documents/types", legaldocuments.GetTypes(deps.Log, deps.PgRepo))
		r.Get("/legal-documents/{id}", legaldocuments.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
		r.Get("/lex-search", lexparser.Search(deps.Log, deps.HTTPClient, deps.Config.LexParser.BaseURL))

		// GES (individual HPP view)
		r.Get("/ges/{id}", gesGet.New(deps.Log, deps.PgRepo))
		r.Get("/ges/{id}/departments", gesDepartments.New(deps.Log, deps.PgRepo))
		r.Get("/ges/{id}/contacts", gesContacts.New(deps.Log, deps.PgRepo, deps.MinioRepo))
		r.Get("/ges/{id}/shutdowns", gesShutdowns.New(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
		r.Get("/ges/{id}/discharges", gesDischarges.New(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
		r.Get("/ges/{id}/incidents", gesIncidents.New(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
		r.Get("/ges/{id}/visits", gesVisits.New(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
		r.Get("/ges/{id}/telemetry", asutpTelemetry.NewGetStation(deps.Log, deps.RedisRepo))
		r.Get("/ges/{id}/telemetry/{device_id}", asutpTelemetry.NewGetDevice(deps.Log, deps.RedisRepo))
		r.Get("/ges/{id}/askue", gesAskue.New(deps.Log, deps.MetricsBlender))

		// Admin routes
		r.Group(func(r chi.Router) {
			r.Use(mwauth.AdminOnly)

			// Roles
			r.Get("/roles", roleGet.New(deps.Log, deps.PgRepo))
			r.Post("/roles", roleAdd.New(deps.Log, deps.PgRepo))
			r.Patch("/roles/{id}", roleEdit.New(deps.Log, deps.PgRepo))
			r.Delete("/roles/{id}", roleDelete.New(deps.Log, deps.PgRepo))

			// Positions
			r.Get("/positions", positionsGet.New(deps.Log, deps.PgRepo))
			r.Post("/positions", positionsAdd.New(deps.Log, deps.PgRepo))
			r.Patch("/positions/{id}", positionsPatch.New(deps.Log, deps.PgRepo))
			r.Delete("/positions/{id}", positionsDelete.New(deps.Log, deps.PgRepo))

			// Users
			r.Get("/users", usersGet.New(deps.Log, deps.PgRepo))
			r.Post("/users", usersAdd.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo))
			r.Patch("/users/{userID}", usersEdit.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo))
			r.Get("/users/{userID}", usersGetById.New(deps.Log, deps.PgRepo))
			r.Delete("/users/{userID}", usersDelete.New(deps.Log, deps.PgRepo))
			r.Post("/users/{userID}/roles", assignRole.New(deps.Log, deps.PgRepo))
			r.Delete("/users/{userID}/roles/{roleID}", revokeRole.New(deps.Log, deps.PgRepo))
		})

		// SC endpoints
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("sc"))

			// Indicator
			r.Put("/indicators/{resID}", setIndicator.New(deps.Log, deps.PgRepo))

			// Upload
			r.Post("/upload/stock", stock.Upload(deps.Log, deps.HTTPClient, deps.Config.Upload.Stock))
			r.Post("/upload/modsnow", table.Upload(deps.Log, deps.HTTPClient, deps.Config.Upload.Modsnow))
			r.Post("/upload/archive", modsnowImg.Upload(deps.Log, deps.HTTPClient, deps.Config.Upload.Archive))
			r.Post("/upload/files", upload.New(deps.Log, deps.MinioRepo, deps.PgRepo, deps.Config.PrimeParser.URL, deps.Config.ApiKey))

			// Reservoirs
			r.Post("/reservoirs", resAdd.New(deps.Log, deps.PgRepo))

			// File category
			r.Get("/files/categories", catGet.New(deps.Log, deps.PgRepo))
			r.Post("/files/categories", catAdd.New(deps.Log, deps.PgRepo))

			// Delete
			r.Delete("/files/{fileID}", fileDelete.New(deps.Log, deps.PgRepo, deps.MinioRepo))
		})

		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("sc", "rais"))

			r.Get("/files/latest", latest.New(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/files/{fileID}/download", download.New(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/files", getbycategory.New(deps.Log, deps.PgRepo, deps.MinioRepo))

			// Discharges (Сбросы)
			r.Get("/discharges", dischargeGet.New(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
			r.Get("/discharges/current", dischargeGetCurrent.New(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/discharges/flat", dischargeGetFlat.New(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
			r.Post("/discharges", dischargeAdd.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/discharges/{id}", dischargePatch.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/discharges/{id}", dischargeDelete.New(deps.Log, deps.PgRepo))
			r.Get("/discharges/export", dischargeExport.New(
				deps.Log,
				deps.PgRepo,
				dischargeExcelGen.New(deps.DischargeExcelTemplatePath),
				loc,
			))

			r.Get("/incidents", incidentsHandler.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
			r.Post("/incidents", incidentsHandler.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/incidents/{id}", incidentsHandler.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/incidents/{id}", incidentsHandler.Delete(deps.Log, deps.PgRepo))

			r.Post("/shutdowns", shutdowns.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/shutdowns/{id}", shutdowns.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/shutdowns/{id}", shutdowns.Delete(deps.Log, deps.PgRepo))
			r.Patch("/shutdowns/{id}/viewed", shutdowns.MarkViewed(deps.Log, deps.PgRepo))

			r.Get("/past-events", pastEventsHandler.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
			r.Get("/past-events/by-type", pastEventsHandler.GetByType(deps.Log, deps.PgRepo, deps.MinioRepo, loc))

			// Calendar
			r.Get("/calendar/events", calendar.Get(deps.Log, deps.PgRepo, loc))

			// Level Volume
			r.Get("/level-volume", levelVolumeGet.New(deps.Log, deps.PgRepo))

			r.Get("/reservoir-device", reservoirdevicesummary.Get(deps.Log, deps.PgRepo))
			r.Patch("/reservoir-device", reservoirdevicesummary.Patch(deps.Log, deps.PgRepo))

			r.Get("/reservoir-summary", reservoirsummary.Get(deps.Log, deps.PgRepo, deps.ReservoirFetcher))
			router.Get("/reservoir-summary/export", reservoirsummary.GetExport(
				deps.Log,
				deps.PgRepo,
				excelgen.New(deps.ExcelTemplatePath),
			))
			r.Post("/reservoir-summary", reservoirsummary.New(deps.Log, deps.PgRepo))

			r.Get("/visits", visit.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
			r.Post("/visits", visit.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/visits/{id}", visit.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/visits/{id}", visit.Delete(deps.Log, deps.PgRepo))

			// SC Export (комплексный суточный отчёт)
			r.Get("/sc/export", scExport.New(
				deps.Log,
				deps.PgRepo, // DischargeGetter
				deps.PgRepo, // ShutdownGetter
				deps.PgRepo, // OrgTypesGetter
				deps.PgRepo, // VisitGetter
				deps.PgRepo, // IncidentGetter
				scExcelGen.New(deps.SCExcelTemplatePath),
				loc,
			))
		})

		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("investment", "rais"))

			// Investment routes
			r.Get("/investments", investments.GetAll(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/investments/{id}", investments.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Post("/investments", investments.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/investments/{id}", investments.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/investments/{id}", investments.Delete(deps.Log, deps.PgRepo))

			// Investment types routes
			r.Get("/investments/types", investments.GetTypes(deps.Log, deps.PgRepo))
			r.Post("/investments/types", investments.AddType(deps.Log, deps.PgRepo))
			r.Patch("/investments/types/{id}", investments.EditType(deps.Log, deps.PgRepo))
			r.Delete("/investments/types/{id}", investments.DeleteType(deps.Log, deps.PgRepo))

			// Investment statuses routes
			r.Get("/investments/statuses", investments.GetStatuses(deps.Log, deps.PgRepo))
			r.Post("/investments/statuses", investments.AddStatus(deps.Log, deps.PgRepo))
			r.Patch("/investments/statuses/{id}", investments.EditStatus(deps.Log, deps.PgRepo))
			r.Delete("/investments/statuses/{id}", investments.DeleteStatus(deps.Log, deps.PgRepo))

			// Invest Active Projects
			r.Get("/invest-active-projects", investActiveProjects.GetAll(deps.Log, deps.PgRepo))
			r.Get("/invest-active-projects/{id}", investActiveProjects.GetByID(deps.Log, deps.PgRepo))
			r.Post("/invest-active-projects", investActiveProjects.Add(deps.Log, deps.PgRepo))
			r.Patch("/invest-active-projects/{id}", investActiveProjects.Edit(deps.Log, deps.PgRepo))
			r.Delete("/invest-active-projects/{id}", investActiveProjects.Delete(deps.Log, deps.PgRepo))
		})

		// Legal Documents (Chancellery - Normative-Legal Library)
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("chancellery", "rais"))

			r.Post("/legal-documents", legaldocuments.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/legal-documents/{id}", legaldocuments.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/legal-documents/{id}", legaldocuments.Delete(deps.Log, deps.PgRepo))
		})

		// Document Workflow (Chancellery - Document Management)
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("chancellery", "rais"))

			// Document Statuses (shared reference)
			r.Get("/document-statuses", docstatuses.GetAll(deps.Log, deps.PgRepo))

			// Decrees (Приказы)
			r.Get("/decrees", decrees.GetAll(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/decrees/types", decrees.GetTypes(deps.Log, deps.PgRepo))
			r.Get("/decrees/{id}", decrees.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/decrees/{id}/history", decrees.GetStatusHistory(deps.Log, deps.PgRepo))
			r.Post("/decrees", decrees.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/decrees/{id}", decrees.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/decrees/{id}/status", decrees.ChangeStatus(deps.Log, deps.PgRepo))
			r.Delete("/decrees/{id}", decrees.Delete(deps.Log, deps.PgRepo))

			// Reports (Рапорты)
			r.Get("/reports", reports.GetAll(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/reports/types", reports.GetTypes(deps.Log, deps.PgRepo))
			r.Get("/reports/{id}", reports.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/reports/{id}/history", reports.GetStatusHistory(deps.Log, deps.PgRepo))
			r.Post("/reports", reports.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/reports/{id}", reports.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/reports/{id}/status", reports.ChangeStatus(deps.Log, deps.PgRepo))
			r.Delete("/reports/{id}", reports.Delete(deps.Log, deps.PgRepo))

			// Letters (Письма)
			r.Get("/letters", letters.GetAll(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/letters/types", letters.GetTypes(deps.Log, deps.PgRepo))
			r.Get("/letters/{id}", letters.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/letters/{id}/history", letters.GetStatusHistory(deps.Log, deps.PgRepo))
			r.Post("/letters", letters.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/letters/{id}", letters.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/letters/{id}/status", letters.ChangeStatus(deps.Log, deps.PgRepo))
			r.Delete("/letters/{id}", letters.Delete(deps.Log, deps.PgRepo))

			// Instructions (Инструкции)
			r.Get("/instructions", instructions.GetAll(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/instructions/types", instructions.GetTypes(deps.Log, deps.PgRepo))
			r.Get("/instructions/{id}", instructions.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/instructions/{id}/history", instructions.GetStatusHistory(deps.Log, deps.PgRepo))
			r.Post("/instructions", instructions.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/instructions/{id}", instructions.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/instructions/{id}/status", instructions.ChangeStatus(deps.Log, deps.PgRepo))
			r.Delete("/instructions/{id}", instructions.Delete(deps.Log, deps.PgRepo))

			// Document Signatures (Подписание документов)
			// Unified list of documents pending signature
			r.Get("/documents/pending-signature", signatures.GetPending(deps.Log, deps.PgRepo))

			// Decrees signatures
			r.Post("/decrees/{id}/sign", signatures.Sign(deps.Log, deps.PgRepo, "decree"))
			r.Post("/decrees/{id}/reject-signature", signatures.Reject(deps.Log, deps.PgRepo, "decree"))
			r.Get("/decrees/{id}/signatures", signatures.GetSignatures(deps.Log, deps.PgRepo, "decree"))

			// Reports signatures
			r.Post("/reports/{id}/sign", signatures.Sign(deps.Log, deps.PgRepo, "report"))
			r.Post("/reports/{id}/reject-signature", signatures.Reject(deps.Log, deps.PgRepo, "report"))
			r.Get("/reports/{id}/signatures", signatures.GetSignatures(deps.Log, deps.PgRepo, "report"))

			// Letters signatures
			r.Post("/letters/{id}/sign", signatures.Sign(deps.Log, deps.PgRepo, "letter"))
			r.Post("/letters/{id}/reject-signature", signatures.Reject(deps.Log, deps.PgRepo, "letter"))
			r.Get("/letters/{id}/signatures", signatures.GetSignatures(deps.Log, deps.PgRepo, "letter"))

			// Instructions signatures
			r.Post("/instructions/{id}/sign", signatures.Sign(deps.Log, deps.PgRepo, "instruction"))
			r.Post("/instructions/{id}/reject-signature", signatures.Reject(deps.Log, deps.PgRepo, "instruction"))
			r.Get("/instructions/{id}/signatures", signatures.GetSignatures(deps.Log, deps.PgRepo, "instruction"))
		})

		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("assistant", "rais"))

			r.Get("/dc", dc.Get(deps.Log, deps.MongoRepo))

			// Events
			r.Get("/events", eventGetAll.New(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Get("/events/short", eventGetShort.New(deps.Log, deps.PgRepo))
			r.Get("/events/statuses", eventGetStatuses.New(deps.Log, deps.PgRepo))
			r.Get("/events/types", eventGetTypes.New(deps.Log, deps.PgRepo))
			r.Get("/events/{id}", eventGetById.New(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Post("/events", eventAdd.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/events/{id}", eventEdit.New(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/events/{id}", eventDelete.New(deps.Log, deps.PgRepo))

			// Fast Calls
			r.Get("/fast-calls", fastCallGetAll.New(deps.Log, deps.PgRepo))
			r.Get("/fast-calls/{id}", fastCallGetById.New(deps.Log, deps.PgRepo))
			r.Post("/fast-calls", fastCallAdd.New(deps.Log, deps.PgRepo))
			r.Patch("/fast-calls/{id}", fastCallEdit.New(deps.Log, deps.PgRepo))
			r.Delete("/fast-calls/{id}", fastCallDelete.New(deps.Log, deps.PgRepo))
		})

		// HRM Module
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("hrm_admin", "hrm_manager", "hrm_employee"))

			r.Route("/hrm", func(r chi.Router) {
				// Dashboard
				r.Get("/dashboard", hrmDashboardHandler.Get(deps.Log, deps.HRMDashboardService))
				r.Patch("/dashboard/notifications/{id}/read", hrmDashboardHandler.MarkRead(deps.Log, deps.HRMDashboardService))
				r.Post("/dashboard/notifications/read-all", hrmDashboardHandler.MarkReadAll(deps.Log, deps.HRMDashboardService))

				// Personnel Records
				r.Get("/personnel-records", hrmPersonnelHandler.GetAll(deps.Log, deps.HRMPersonnelService))
				r.Post("/personnel-records", hrmPersonnelHandler.Create(deps.Log, deps.HRMPersonnelService))
				r.Get("/personnel-records/employee/{id}", hrmPersonnelHandler.GetByEmployee(deps.Log, deps.HRMPersonnelService))
				r.Get("/personnel-records/{id}", hrmPersonnelHandler.GetByID(deps.Log, deps.HRMPersonnelService))
				r.Patch("/personnel-records/{id}", hrmPersonnelHandler.Update(deps.Log, deps.HRMPersonnelService))
				r.Delete("/personnel-records/{id}", hrmPersonnelHandler.Delete(deps.Log, deps.HRMPersonnelService))
				r.Get("/personnel-records/{id}/documents", hrmPersonnelHandler.GetDocuments(deps.Log, deps.HRMPersonnelService))
				r.Get("/personnel-records/{id}/transfers", hrmPersonnelHandler.GetTransfers(deps.Log, deps.HRMPersonnelService))

				// Vacations — register specific routes BEFORE {id} routes
				r.Get("/vacations/calendar", hrmVacationHandler.GetCalendar(deps.Log, deps.HRMVacationService))
				r.Get("/vacations/balances", hrmVacationHandler.GetBalances(deps.Log, deps.HRMVacationService))
				r.Get("/vacations/pending", hrmVacationHandler.GetPending(deps.Log, deps.HRMVacationService))
				r.Get("/vacations/balance/{id}", hrmVacationHandler.GetBalance(deps.Log, deps.HRMVacationService))
				r.Get("/vacations", hrmVacationHandler.GetAll(deps.Log, deps.HRMVacationService))
				r.Post("/vacations", hrmVacationHandler.Create(deps.Log, deps.HRMVacationService))
				r.Get("/vacations/{id}", hrmVacationHandler.GetByID(deps.Log, deps.HRMVacationService))
				r.Patch("/vacations/{id}", hrmVacationHandler.Update(deps.Log, deps.HRMVacationService))
				r.Delete("/vacations/{id}", hrmVacationHandler.Delete(deps.Log, deps.HRMVacationService))
				r.Post("/vacations/{id}/approve", hrmVacationHandler.Approve(deps.Log, deps.HRMVacationService))
				r.Post("/vacations/{id}/reject", hrmVacationHandler.Reject(deps.Log, deps.HRMVacationService))
				r.Post("/vacations/{id}/cancel", hrmVacationHandler.Cancel(deps.Log, deps.HRMVacationService))
			})
		})

		// Receptions
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("sc", "assistant", "rais"))

			r.Get("/receptions", receptionGetAll.New(deps.Log, deps.PgRepo, loc))
			r.Get("/receptions/{id}", receptionGetById.New(deps.Log, deps.PgRepo))
			r.Post("/receptions", receptionAdd.New(deps.Log, deps.PgRepo))
			r.Patch("/receptions/{id}", receptionEdit.New(deps.Log, deps.PgRepo))
			r.Delete("/receptions/{id}", receptionDelete.New(deps.Log, deps.PgRepo))
		})
	})

	// MODSNOW Pipeline API
	router.Route("/api/snow-cover", func(r chi.Router) {
		r.Use(asutpauth.RequireToken(deps.Config.ModsnowToken))
		r.Post("/", snowCover.New(deps.Log, deps.PgRepo, loc))
	})

	// ASUTP Telemetry API - POST with Bearer token (for GES systems)
	router.Route("/api/v1/asutp", func(r chi.Router) {
		r.Use(asutpauth.RequireToken(deps.Config.ASUTP.Token))

		r.Post("/telemetry/{station_db_id}", asutpTelemetry.NewPost(deps.Log, deps.RedisRepo, deps.AlarmProcessor))
	})
}
