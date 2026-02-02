package router

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
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
	hrmAccess "srmt-admin/internal/http-server/handlers/hrm/access"
	hrmAnalytics "srmt-admin/internal/http-server/handlers/hrm/analytics"
	hrmCabinet "srmt-admin/internal/http-server/handlers/hrm/cabinet"
	hrmCompetency "srmt-admin/internal/http-server/handlers/hrm/competency"
	hrmDashboard "srmt-admin/internal/http-server/handlers/hrm/dashboard"
	hrmDocument "srmt-admin/internal/http-server/handlers/hrm/document"
	hrmEmployeeAdd "srmt-admin/internal/http-server/handlers/hrm/employee/add"
	hrmEmployeeDelete "srmt-admin/internal/http-server/handlers/hrm/employee/delete"
	hrmEmployeeEdit "srmt-admin/internal/http-server/handlers/hrm/employee/edit"
	hrmEmployeeGet "srmt-admin/internal/http-server/handlers/hrm/employee/get"
	hrmEmployeeGetByID "srmt-admin/internal/http-server/handlers/hrm/employee/get-by-id"
	hrmEmployeeTerminate "srmt-admin/internal/http-server/handlers/hrm/employee/terminate"
	hrmNotification "srmt-admin/internal/http-server/handlers/hrm/notification"
	hrmPerformance "srmt-admin/internal/http-server/handlers/hrm/performance"
	hrmPersonnelDocAdd "srmt-admin/internal/http-server/handlers/hrm/personnel-document/add"
	hrmPersonnelDocDelete "srmt-admin/internal/http-server/handlers/hrm/personnel-document/delete"
	hrmPersonnelDocEdit "srmt-admin/internal/http-server/handlers/hrm/personnel-document/edit"
	hrmPersonnelDocGet "srmt-admin/internal/http-server/handlers/hrm/personnel-document/get"
	hrmRecruiting "srmt-admin/internal/http-server/handlers/hrm/recruiting"
	hrmSalary "srmt-admin/internal/http-server/handlers/hrm/salary"
	hrmTimesheet "srmt-admin/internal/http-server/handlers/hrm/timesheet"
	hrmTraining "srmt-admin/internal/http-server/handlers/hrm/training"
	hrmTransferAdd "srmt-admin/internal/http-server/handlers/hrm/transfer/add"
	hrmTransferGet "srmt-admin/internal/http-server/handlers/hrm/transfer/get"
	hrmVacation "srmt-admin/internal/http-server/handlers/hrm/vacation"
	incidentsHandler "srmt-admin/internal/http-server/handlers/incidents-handler"
	setIndicator "srmt-admin/internal/http-server/handlers/indicators/set"
	"srmt-admin/internal/http-server/handlers/instructions"
	investActiveProjects "srmt-admin/internal/http-server/handlers/invest-active-projects"
	"srmt-admin/internal/http-server/handlers/investments"
	legaldocuments "srmt-admin/internal/http-server/handlers/legal-documents"
	"srmt-admin/internal/http-server/handlers/letters"
	levelVolumeGet "srmt-admin/internal/http-server/handlers/level-volume/get"
	lexparser "srmt-admin/internal/http-server/handlers/lex-parser"
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
	modsnowImg "srmt-admin/internal/http-server/handlers/sc/modsnow/img"
	"srmt-admin/internal/http-server/handlers/sc/modsnow/table"
	"srmt-admin/internal/http-server/handlers/sc/stock"
	"srmt-admin/internal/http-server/handlers/shutdowns"
	"srmt-admin/internal/http-server/handlers/signatures"
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
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	"srmt-admin/internal/lib/service/ascue"
	excelgen "srmt-admin/internal/lib/service/excel/reservoir-summary"
	"srmt-admin/internal/lib/service/reservoir"
	"srmt-admin/internal/storage/minio"
	"srmt-admin/internal/storage/mongo"
	"srmt-admin/internal/storage/repo"
	"srmt-admin/internal/token"
	"time"

	"github.com/go-chi/chi/v5"
)

// AppDependencies contains all dependencies needed for route setup
type AppDependencies struct {
	Log               *slog.Logger
	Token             *token.Token
	PgRepo            *repo.Repo
	MongoRepo         *mongo.Repo
	MinioRepo         *minio.Repo
	Config            config.Config
	Location          *time.Location
	ASCUEFetcher      *ascue.Fetcher
	ReservoirFetcher  *reservoir.Fetcher
	HTTPClient        *http.Client
	ExcelTemplatePath string
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

		// Employee Cabinet (Personal workspace - available to all authenticated users)
		r.Route("/my-profile", func(r chi.Router) {
			r.Get("/", hrmCabinet.GetProfile(deps.Log, deps.PgRepo))
			r.Patch("/", hrmCabinet.UpdateProfile(deps.Log, deps.PgRepo))
		})

		r.Get("/my-leave-balance", hrmCabinet.GetLeaveBalance(deps.Log, deps.PgRepo))

		r.Route("/my-vacations", func(r chi.Router) {
			r.Get("/", hrmCabinet.GetMyVacations(deps.Log, deps.PgRepo))
			r.Post("/", hrmCabinet.CreateMyVacation(deps.Log, deps.PgRepo))
			r.Post("/{id}/cancel", hrmCabinet.CancelMyVacation(deps.Log, deps.PgRepo))
		})

		r.Route("/my-salary", func(r chi.Router) {
			r.Get("/", hrmCabinet.GetMySalary(deps.Log, deps.PgRepo))
			r.Get("/payslip/{id}", hrmCabinet.GetMyPayslip(deps.Log, deps.PgRepo))
		})

		r.Get("/my-training", hrmCabinet.GetMyTraining(deps.Log, deps.PgRepo))
		r.Get("/my-competencies", hrmCabinet.GetMyCompetencies(deps.Log, deps.PgRepo))

		r.Route("/my-notifications", func(r chi.Router) {
			r.Get("/", hrmCabinet.GetMyNotifications(deps.Log, deps.PgRepo))
			r.Patch("/{id}/read", hrmCabinet.MarkNotificationRead(deps.Log, deps.PgRepo))
			r.Post("/read-all", hrmCabinet.MarkAllNotificationsRead(deps.Log, deps.PgRepo))
		})

		r.Get("/my-tasks", hrmCabinet.GetMyTasks(deps.Log, deps.PgRepo))

		r.Route("/my-documents", func(r chi.Router) {
			r.Get("/", hrmCabinet.GetMyDocuments(deps.Log, deps.PgRepo))
			r.Get("/{id}/download", hrmCabinet.DownloadMyDocument(deps.Log, deps.PgRepo))
		})

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
		r.Get("/dashboard/cascades", orgGetCascades.New(deps.Log, deps.PgRepo, deps.ASCUEFetcher))
		r.Get("/dashboard/production", production.New(deps.Log, deps.PgRepo))
		r.Get("/dashboard/production-stats", productionstats.New(deps.Log, deps.PgRepo))

		// Open routes (available to all authenticated users)
		r.Get("/shutdowns", shutdowns.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
		r.Get("/legal-documents", legaldocuments.GetAll(deps.Log, deps.PgRepo, deps.MinioRepo))
		r.Get("/legal-documents/types", legaldocuments.GetTypes(deps.Log, deps.PgRepo))
		r.Get("/legal-documents/{id}", legaldocuments.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
		r.Get("/lex-search", lexparser.Search(deps.Log, deps.HTTPClient, deps.Config.LexParser.BaseURL))

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

			r.Get("/incidents", incidentsHandler.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
			r.Post("/incidents", incidentsHandler.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/incidents/{id}", incidentsHandler.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/incidents/{id}", incidentsHandler.Delete(deps.Log, deps.PgRepo))

			r.Post("/shutdowns", shutdowns.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/shutdowns/{id}", shutdowns.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/shutdowns/{id}", shutdowns.Delete(deps.Log, deps.PgRepo))

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

		// Receptions
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("sc", "assistant", "rais"))

			r.Get("/receptions", receptionGetAll.New(deps.Log, deps.PgRepo, loc))
			r.Get("/receptions/{id}", receptionGetById.New(deps.Log, deps.PgRepo))
			r.Post("/receptions", receptionAdd.New(deps.Log, deps.PgRepo))
			r.Patch("/receptions/{id}", receptionEdit.New(deps.Log, deps.PgRepo))
			r.Delete("/receptions/{id}", receptionDelete.New(deps.Log, deps.PgRepo))
		})

		// HRM Module - Human Resource Management
		r.Group(func(r chi.Router) {
			r.Use(mwauth.RequireAnyRole("hrm", "rais"))

			r.Route("/hrm", func(r chi.Router) {
				// Dashboard
				r.Get("/dashboard", hrmDashboard.Get(deps.Log, deps.PgRepo))

				// Analytics
				r.Route("/analytics", func(r chi.Router) {
					r.Get("/dashboard", hrmAnalytics.GetDashboard(deps.Log, deps.PgRepo))

					r.Route("/reports", func(r chi.Router) {
						r.Get("/headcount", hrmAnalytics.GetHeadcountReport(deps.Log, deps.PgRepo))
						r.Get("/headcount-trend", hrmAnalytics.GetHeadcountTrend(deps.Log, deps.PgRepo))
						r.Get("/turnover", hrmAnalytics.GetTurnoverReport(deps.Log, deps.PgRepo))
						r.Get("/turnover-trend", hrmAnalytics.GetTurnoverTrend(deps.Log, deps.PgRepo))
						r.Get("/attendance", hrmAnalytics.GetAttendanceReport(deps.Log, deps.PgRepo))
						r.Get("/salary", hrmAnalytics.GetSalaryReport(deps.Log, deps.PgRepo))
						r.Get("/salary-trend", hrmAnalytics.GetSalaryTrend(deps.Log, deps.PgRepo))
						r.Get("/performance", hrmAnalytics.GetPerformanceReport(deps.Log, deps.PgRepo))
						r.Get("/training", hrmAnalytics.GetTrainingReport(deps.Log, deps.PgRepo))
						r.Get("/demographics", hrmAnalytics.GetDemographicsReport(deps.Log, deps.PgRepo))
						r.Post("/custom", hrmAnalytics.GenerateCustomReport(deps.Log, deps.PgRepo))
					})

					r.Route("/export", func(r chi.Router) {
						r.Post("/", hrmAnalytics.Export(deps.Log, deps.PgRepo))
						r.Post("/pdf", hrmAnalytics.ExportPDF(deps.Log, deps.PgRepo))
						r.Post("/excel", hrmAnalytics.ExportExcel(deps.Log, deps.PgRepo))
					})
				})

				// Employees
				r.Route("/employees", func(r chi.Router) {
					r.Get("/", hrmEmployeeGet.New(deps.Log, deps.PgRepo))
					r.Post("/", hrmEmployeeAdd.New(deps.Log, deps.PgRepo))
					r.Get("/{id}", hrmEmployeeGetByID.New(deps.Log, deps.PgRepo))
					r.Patch("/{id}", hrmEmployeeEdit.New(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmEmployeeDelete.New(deps.Log, deps.PgRepo))
					r.Post("/{id}/terminate", hrmEmployeeTerminate.New(deps.Log, deps.PgRepo))
				})

				// Personnel Documents
				r.Route("/personnel-documents", func(r chi.Router) {
					r.Get("/", hrmPersonnelDocGet.New(deps.Log, deps.PgRepo))
					r.Post("/", hrmPersonnelDocAdd.New(deps.Log, deps.PgRepo))
					r.Patch("/{id}", hrmPersonnelDocEdit.New(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmPersonnelDocDelete.New(deps.Log, deps.PgRepo))
				})

				// Transfers
				r.Route("/transfers", func(r chi.Router) {
					r.Get("/", hrmTransferGet.New(deps.Log, deps.PgRepo))
					r.Post("/", hrmTransferAdd.New(deps.Log, deps.PgRepo))
				})

				// Vacations
				r.Route("/vacations", func(r chi.Router) {
					// Vacation Types
					r.Get("/types", hrmVacation.GetTypes(deps.Log, deps.PgRepo))
					r.Post("/types", hrmVacation.AddType(deps.Log, deps.PgRepo))
					r.Patch("/types/{id}", hrmVacation.EditType(deps.Log, deps.PgRepo))
					r.Delete("/types/{id}", hrmVacation.DeleteType(deps.Log, deps.PgRepo))

					// Vacation Balances
					r.Get("/balances", hrmVacation.GetBalances(deps.Log, deps.PgRepo))
					r.Post("/balances", hrmVacation.AddBalance(deps.Log, deps.PgRepo))
					r.Patch("/balances/{id}", hrmVacation.EditBalance(deps.Log, deps.PgRepo))

					// Vacation Requests
					r.Get("/", hrmVacation.GetAll(deps.Log, deps.PgRepo, deps.PgRepo))
					r.Post("/", hrmVacation.Add(deps.Log, deps.PgRepo))
					r.Get("/calendar", hrmVacation.GetCalendar(deps.Log, deps.PgRepo))
					r.Get("/{id}", hrmVacation.GetByID(deps.Log, deps.PgRepo, deps.PgRepo))
					r.Patch("/{id}", hrmVacation.Edit(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmVacation.Delete(deps.Log, deps.PgRepo))
					r.Post("/{id}/approve", hrmVacation.Approve(deps.Log, deps.PgRepo))
					r.Post("/{id}/cancel", hrmVacation.Cancel(deps.Log, deps.PgRepo))
				})

				// Salaries - requires hr, manager, or admin role for write operations
				r.Route("/salaries", func(r chi.Router) {
					// Salary Structures - read is allowed for all HRM users, write requires elevated roles
					r.Get("/structures", hrmSalary.GetStructures(deps.Log, deps.PgRepo, deps.PgRepo))
					r.Group(func(r chi.Router) {
						r.Use(mwauth.RequireAnyRole("hr", "admin"))
						r.Post("/structures", hrmSalary.AddStructure(deps.Log, deps.PgRepo))
						r.Patch("/structures/{id}", hrmSalary.EditStructure(deps.Log, deps.PgRepo))
						r.Delete("/structures/{id}", hrmSalary.DeleteStructure(deps.Log, deps.PgRepo))
					})

					// Salaries - read has row-level access control, write requires elevated roles
					r.Get("/", hrmSalary.GetAll(deps.Log, deps.PgRepo, deps.PgRepo))
					r.Get("/{id}", hrmSalary.GetByID(deps.Log, deps.PgRepo, deps.PgRepo))
					r.Group(func(r chi.Router) {
						r.Use(mwauth.RequireAnyRole("hr", "admin"))
						r.Post("/", hrmSalary.Add(deps.Log, deps.PgRepo))
						r.Delete("/{id}", hrmSalary.Delete(deps.Log, deps.PgRepo))
						r.Post("/{id}/approve", hrmSalary.Approve(deps.Log, deps.PgRepo))
						r.Post("/{id}/pay", hrmSalary.Pay(deps.Log, deps.PgRepo))
					})

					// Bonuses
					r.Get("/bonuses", hrmSalary.GetBonuses(deps.Log, deps.PgRepo))
					r.Post("/bonuses", hrmSalary.AddBonus(deps.Log, deps.PgRepo))
					r.Delete("/bonuses/{id}", hrmSalary.DeleteBonus(deps.Log, deps.PgRepo))
					r.Post("/bonuses/{id}/approve", hrmSalary.ApproveBonus(deps.Log, deps.PgRepo))

					// Deductions
					r.Get("/deductions", hrmSalary.GetDeductions(deps.Log, deps.PgRepo))
					r.Post("/deductions", hrmSalary.AddDeduction(deps.Log, deps.PgRepo))
					r.Delete("/deductions/{id}", hrmSalary.DeleteDeduction(deps.Log, deps.PgRepo))
				})

				// Timesheets
				r.Route("/timesheets", func(r chi.Router) {
					// Holidays
					r.Get("/holidays", hrmTimesheet.GetHolidays(deps.Log, deps.PgRepo))
					r.Post("/holidays", hrmTimesheet.AddHoliday(deps.Log, deps.PgRepo))
					r.Delete("/holidays/{id}", hrmTimesheet.DeleteHoliday(deps.Log, deps.PgRepo))

					// Timesheet Entries
					r.Get("/entries", hrmTimesheet.GetEntries(deps.Log, deps.PgRepo))
					r.Post("/entries", hrmTimesheet.AddEntry(deps.Log, deps.PgRepo))
					r.Patch("/entries/{id}", hrmTimesheet.EditEntry(deps.Log, deps.PgRepo))
					r.Delete("/entries/{id}", hrmTimesheet.DeleteEntry(deps.Log, deps.PgRepo))

					// Timesheets
					r.Get("/", hrmTimesheet.GetTimesheets(deps.Log, deps.PgRepo))
					r.Post("/", hrmTimesheet.AddTimesheet(deps.Log, deps.PgRepo))
					r.Get("/{id}", hrmTimesheet.GetTimesheetByID(deps.Log, deps.PgRepo))
					r.Patch("/{id}", hrmTimesheet.EditTimesheet(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmTimesheet.DeleteTimesheet(deps.Log, deps.PgRepo))
					r.Post("/{id}/submit", hrmTimesheet.SubmitTimesheet(deps.Log, deps.PgRepo))
					r.Post("/{id}/approve", hrmTimesheet.ApproveTimesheet(deps.Log, deps.PgRepo))
				})

				// Recruiting
				r.Route("/recruiting", func(r chi.Router) {
					// Vacancies
					r.Get("/vacancies", hrmRecruiting.GetVacancies(deps.Log, deps.PgRepo))
					r.Post("/vacancies", hrmRecruiting.AddVacancy(deps.Log, deps.PgRepo))
					r.Get("/vacancies/{id}", hrmRecruiting.GetVacancyByID(deps.Log, deps.PgRepo))
					r.Patch("/vacancies/{id}", hrmRecruiting.EditVacancy(deps.Log, deps.PgRepo))
					r.Delete("/vacancies/{id}", hrmRecruiting.DeleteVacancy(deps.Log, deps.PgRepo))
					r.Post("/vacancies/{id}/publish", hrmRecruiting.PublishVacancy(deps.Log, deps.PgRepo))
					r.Post("/vacancies/{id}/close", hrmRecruiting.CloseVacancy(deps.Log, deps.PgRepo))

					// Candidates
					r.Get("/candidates", hrmRecruiting.GetCandidates(deps.Log, deps.PgRepo))
					r.Post("/candidates", hrmRecruiting.AddCandidate(deps.Log, deps.PgRepo))
					r.Get("/candidates/{id}", hrmRecruiting.GetCandidateByID(deps.Log, deps.PgRepo))
					r.Patch("/candidates/{id}", hrmRecruiting.EditCandidate(deps.Log, deps.PgRepo))
					r.Delete("/candidates/{id}", hrmRecruiting.DeleteCandidate(deps.Log, deps.PgRepo))
					r.Post("/candidates/{id}/status", hrmRecruiting.MoveCandidateStatus(deps.Log, deps.PgRepo))

					// Interviews
					r.Get("/interviews", hrmRecruiting.GetInterviews(deps.Log, deps.PgRepo))
					r.Post("/interviews", hrmRecruiting.AddInterview(deps.Log, deps.PgRepo))
					r.Get("/interviews/{id}", hrmRecruiting.GetInterviewByID(deps.Log, deps.PgRepo))
					r.Patch("/interviews/{id}", hrmRecruiting.EditInterview(deps.Log, deps.PgRepo))
					r.Delete("/interviews/{id}", hrmRecruiting.DeleteInterview(deps.Log, deps.PgRepo))
					r.Post("/interviews/{id}/complete", hrmRecruiting.CompleteInterview(deps.Log, deps.PgRepo))
					r.Post("/interviews/{id}/cancel", hrmRecruiting.CancelInterview(deps.Log, deps.PgRepo))
				})

				// Training
				r.Route("/trainings", func(r chi.Router) {
					// Trainings
					r.Get("/", hrmTraining.GetTrainings(deps.Log, deps.PgRepo))
					r.Post("/", hrmTraining.AddTraining(deps.Log, deps.PgRepo))
					r.Get("/{id}", hrmTraining.GetTrainingByID(deps.Log, deps.PgRepo))
					r.Patch("/{id}", hrmTraining.EditTraining(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmTraining.DeleteTraining(deps.Log, deps.PgRepo))

					// Participants
					r.Get("/participants", hrmTraining.GetParticipants(deps.Log, deps.PgRepo))
					r.Post("/participants", hrmTraining.EnrollParticipant(deps.Log, deps.PgRepo))
					r.Post("/participants/{id}/complete", hrmTraining.CompleteParticipantTraining(deps.Log, deps.PgRepo))

					// Certificates
					r.Get("/certificates", hrmTraining.GetCertificates(deps.Log, deps.PgRepo))
					r.Post("/certificates", hrmTraining.AddCertificate(deps.Log, deps.PgRepo))
					r.Patch("/certificates/{id}", hrmTraining.EditCertificate(deps.Log, deps.PgRepo))
					r.Delete("/certificates/{id}", hrmTraining.DeleteCertificate(deps.Log, deps.PgRepo))
				})

				// Competencies
				r.Route("/competencies", func(r chi.Router) {
					// Categories
					r.Get("/categories", hrmCompetency.GetCategories(deps.Log, deps.PgRepo))
					r.Post("/categories", hrmCompetency.AddCategory(deps.Log, deps.PgRepo))
					r.Get("/categories/{id}", hrmCompetency.GetCategoryByID(deps.Log, deps.PgRepo))
					r.Patch("/categories/{id}", hrmCompetency.EditCategory(deps.Log, deps.PgRepo))
					r.Delete("/categories/{id}", hrmCompetency.DeleteCategory(deps.Log, deps.PgRepo))

					// Competencies
					r.Get("/", hrmCompetency.GetCompetencies(deps.Log, deps.PgRepo))
					r.Post("/", hrmCompetency.AddCompetency(deps.Log, deps.PgRepo))
					r.Get("/{id}", hrmCompetency.GetCompetencyByID(deps.Log, deps.PgRepo))
					r.Patch("/{id}", hrmCompetency.EditCompetency(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmCompetency.DeleteCompetency(deps.Log, deps.PgRepo))

					// Levels
					r.Get("/{competencyId}/levels", hrmCompetency.GetLevels(deps.Log, deps.PgRepo))
					r.Post("/levels", hrmCompetency.AddLevel(deps.Log, deps.PgRepo))
					r.Patch("/levels/{id}", hrmCompetency.EditLevel(deps.Log, deps.PgRepo))
					r.Delete("/levels/{id}", hrmCompetency.DeleteLevel(deps.Log, deps.PgRepo))

					// Matrix
					r.Get("/matrix", hrmCompetency.GetMatrix(deps.Log, deps.PgRepo))
					r.Post("/matrix", hrmCompetency.AddMatrix(deps.Log, deps.PgRepo))
					r.Patch("/matrix/{id}", hrmCompetency.EditMatrix(deps.Log, deps.PgRepo))
					r.Delete("/matrix/{id}", hrmCompetency.DeleteMatrix(deps.Log, deps.PgRepo))

					// Assessments
					r.Get("/assessments", hrmCompetency.GetAssessments(deps.Log, deps.PgRepo))
					r.Post("/assessments", hrmCompetency.AddAssessment(deps.Log, deps.PgRepo))
					r.Get("/assessments/{id}", hrmCompetency.GetAssessmentByID(deps.Log, deps.PgRepo))
					r.Post("/assessments/{id}/start", hrmCompetency.StartAssessment(deps.Log, deps.PgRepo))
					r.Post("/assessments/{id}/complete", hrmCompetency.CompleteAssessment(deps.Log, deps.PgRepo))
					r.Delete("/assessments/{id}", hrmCompetency.DeleteAssessment(deps.Log, deps.PgRepo))

					// Scores
					r.Get("/scores", hrmCompetency.GetScores(deps.Log, deps.PgRepo))
					r.Post("/scores", hrmCompetency.AddScore(deps.Log, deps.PgRepo))
					r.Post("/scores/bulk", hrmCompetency.BulkAddScores(deps.Log, deps.PgRepo))
					r.Patch("/scores/{id}", hrmCompetency.EditScore(deps.Log, deps.PgRepo))
					r.Delete("/scores/{id}", hrmCompetency.DeleteScore(deps.Log, deps.PgRepo))
				})

				// Performance
				r.Route("/performance", func(r chi.Router) {
					// Reviews
					r.Get("/reviews", hrmPerformance.GetReviews(deps.Log, deps.PgRepo))
					r.Post("/reviews", hrmPerformance.AddReview(deps.Log, deps.PgRepo))
					r.Get("/reviews/{id}", hrmPerformance.GetReviewByID(deps.Log, deps.PgRepo))
					r.Patch("/reviews/{id}", hrmPerformance.EditReview(deps.Log, deps.PgRepo))
					r.Delete("/reviews/{id}", hrmPerformance.DeleteReview(deps.Log, deps.PgRepo))
					r.Post("/reviews/{id}/self-review", hrmPerformance.SubmitSelfReview(deps.Log, deps.PgRepo))
					r.Post("/reviews/{id}/manager-review", hrmPerformance.SubmitManagerReview(deps.Log, deps.PgRepo))
					r.Post("/reviews/{id}/calibrate", hrmPerformance.CalibrateReview(deps.Log, deps.PgRepo))

					// Goals
					r.Get("/goals", hrmPerformance.GetGoals(deps.Log, deps.PgRepo))
					r.Post("/goals", hrmPerformance.AddGoal(deps.Log, deps.PgRepo))
					r.Get("/goals/{id}", hrmPerformance.GetGoalByID(deps.Log, deps.PgRepo))
					r.Patch("/goals/{id}", hrmPerformance.EditGoal(deps.Log, deps.PgRepo))
					r.Delete("/goals/{id}", hrmPerformance.DeleteGoal(deps.Log, deps.PgRepo))
					r.Post("/goals/{id}/progress", hrmPerformance.UpdateGoalProgress(deps.Log, deps.PgRepo))
					r.Post("/goals/{id}/rate", hrmPerformance.RateGoal(deps.Log, deps.PgRepo))

					// KPIs
					r.Get("/kpis", hrmPerformance.GetKPIs(deps.Log, deps.PgRepo))
					r.Post("/kpis", hrmPerformance.AddKPI(deps.Log, deps.PgRepo))
					r.Get("/kpis/{id}", hrmPerformance.GetKPIByID(deps.Log, deps.PgRepo))
					r.Patch("/kpis/{id}", hrmPerformance.EditKPI(deps.Log, deps.PgRepo))
					r.Delete("/kpis/{id}", hrmPerformance.DeleteKPI(deps.Log, deps.PgRepo))
					r.Post("/kpis/{id}/value", hrmPerformance.UpdateKPIValue(deps.Log, deps.PgRepo))
					r.Post("/kpis/{id}/rate", hrmPerformance.RateKPI(deps.Log, deps.PgRepo))
				})

				// Documents
				r.Route("/documents", func(r chi.Router) {
					// Document Types
					r.Get("/types", hrmDocument.GetTypes(deps.Log, deps.PgRepo))
					r.Post("/types", hrmDocument.AddType(deps.Log, deps.PgRepo))
					r.Get("/types/{id}", hrmDocument.GetTypeByID(deps.Log, deps.PgRepo))
					r.Patch("/types/{id}", hrmDocument.EditType(deps.Log, deps.PgRepo))
					r.Delete("/types/{id}", hrmDocument.DeleteType(deps.Log, deps.PgRepo))

					// Documents
					r.Get("/", hrmDocument.GetDocuments(deps.Log, deps.PgRepo))
					r.Post("/", hrmDocument.AddDocument(deps.Log, deps.PgRepo))
					r.Get("/{id}", hrmDocument.GetDocumentByID(deps.Log, deps.PgRepo))
					r.Patch("/{id}", hrmDocument.EditDocument(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmDocument.DeleteDocument(deps.Log, deps.PgRepo))

					// Signatures
					r.Get("/signatures", hrmDocument.GetSignatures(deps.Log, deps.PgRepo))
					r.Post("/signatures", hrmDocument.AddSignature(deps.Log, deps.PgRepo))
					r.Post("/signatures/{id}/sign", hrmDocument.SignDocument(deps.Log, deps.PgRepo))

					// Templates
					r.Get("/templates", hrmDocument.GetTemplates(deps.Log, deps.PgRepo))
					r.Post("/templates", hrmDocument.AddTemplate(deps.Log, deps.PgRepo))
					r.Get("/templates/{id}", hrmDocument.GetTemplateByID(deps.Log, deps.PgRepo))
					r.Patch("/templates/{id}", hrmDocument.EditTemplate(deps.Log, deps.PgRepo))
					r.Delete("/templates/{id}", hrmDocument.DeleteTemplate(deps.Log, deps.PgRepo))
				})

				// Access Control - requires security or admin role
				r.Route("/access", func(r chi.Router) {
					r.Use(mwauth.RequireAnyRole("security", "admin", "hr"))

					// Zones
					r.Get("/zones", hrmAccess.GetZones(deps.Log, deps.PgRepo))
					r.Post("/zones", hrmAccess.AddZone(deps.Log, deps.PgRepo))
					r.Get("/zones/{id}", hrmAccess.GetZoneByID(deps.Log, deps.PgRepo))
					r.Patch("/zones/{id}", hrmAccess.EditZone(deps.Log, deps.PgRepo))
					r.Delete("/zones/{id}", hrmAccess.DeleteZone(deps.Log, deps.PgRepo))

					// Cards
					r.Get("/cards", hrmAccess.GetCards(deps.Log, deps.PgRepo))
					r.Post("/cards", hrmAccess.AddCard(deps.Log, deps.PgRepo))
					r.Get("/cards/{id}", hrmAccess.GetCardByID(deps.Log, deps.PgRepo))
					r.Patch("/cards/{id}", hrmAccess.EditCard(deps.Log, deps.PgRepo))
					r.Delete("/cards/{id}", hrmAccess.DeleteCard(deps.Log, deps.PgRepo))
					r.Post("/cards/{id}/deactivate", hrmAccess.DeactivateCard(deps.Log, deps.PgRepo))

					// Card Zone Access
					r.Get("/card-zones", hrmAccess.GetCardZoneAccess(deps.Log, deps.PgRepo))
					r.Post("/card-zones", hrmAccess.AddCardZoneAccess(deps.Log, deps.PgRepo))
					r.Delete("/card-zones/{id}", hrmAccess.DeleteCardZoneAccess(deps.Log, deps.PgRepo))

					// Access Logs - additional security: only security and admin can view
					r.Group(func(r chi.Router) {
						r.Use(mwauth.RequireAnyRole("security", "admin"))
						r.Get("/logs", hrmAccess.GetAccessLogs(deps.Log, deps.PgRepo))
						r.Post("/logs", hrmAccess.AddAccessLog(deps.Log, deps.PgRepo))
					})
				})

				// Notifications
				r.Route("/notifications", func(r chi.Router) {
					r.Get("/", hrmNotification.GetNotifications(deps.Log, deps.PgRepo))
					r.Post("/", hrmNotification.AddNotification(deps.Log, deps.PgRepo))
					r.Get("/unread-count", hrmNotification.GetUnreadCount(deps.Log, deps.PgRepo))
					r.Post("/mark-all-read", hrmNotification.MarkAllAsRead(deps.Log, deps.PgRepo))
					r.Get("/{id}", hrmNotification.GetNotificationByID(deps.Log, deps.PgRepo))
					r.Post("/{id}/read", hrmNotification.MarkAsRead(deps.Log, deps.PgRepo))
					r.Delete("/{id}", hrmNotification.DeleteNotification(deps.Log, deps.PgRepo))
				})
			})
		})
	})
}
