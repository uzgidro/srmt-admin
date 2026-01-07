package router

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/handlers/auth/me"
	"srmt-admin/internal/http-server/handlers/auth/refresh"
	signIn "srmt-admin/internal/http-server/handlers/auth/sign-in"
	signOut "srmt-admin/internal/http-server/handlers/auth/sign-out"
	contactAdd "srmt-admin/internal/http-server/handlers/contacts/add"
	contactDelete "srmt-admin/internal/http-server/handlers/contacts/delete"
	contactEdit "srmt-admin/internal/http-server/handlers/contacts/edit"
	contactGetAll "srmt-admin/internal/http-server/handlers/contacts/get-all"
	contactGetById "srmt-admin/internal/http-server/handlers/contacts/get-by-id"
	"srmt-admin/internal/http-server/handlers/currency"
	orgGetCascades "srmt-admin/internal/http-server/handlers/dashboard/get-cascades"
	dashboardGetReservoir "srmt-admin/internal/http-server/handlers/dashboard/get-reservoir"
	dashboardGetReservoirHourly "srmt-admin/internal/http-server/handlers/dashboard/get-reservoir-hourly"
	"srmt-admin/internal/http-server/handlers/data/analytics"
	dataSet "srmt-admin/internal/http-server/handlers/data/set"
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
	incidentsHandler "srmt-admin/internal/http-server/handlers/incidents-handler"
	setIndicator "srmt-admin/internal/http-server/handlers/indicators/set"
	investActiveProjects "srmt-admin/internal/http-server/handlers/invest-active-projects"
	"srmt-admin/internal/http-server/handlers/investments"
	levelVolumeGet "srmt-admin/internal/http-server/handlers/level-volume/get"
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
	reservoirdevicesummary "srmt-admin/internal/http-server/handlers/reservoir-device-summary"
	reservoirsummary "srmt-admin/internal/http-server/handlers/reservoir-summary"
	resAdd "srmt-admin/internal/http-server/handlers/reservoirs/add"
	roleAdd "srmt-admin/internal/http-server/handlers/role/add"
	roleDelete "srmt-admin/internal/http-server/handlers/role/delete"
	roleEdit "srmt-admin/internal/http-server/handlers/role/edit"
	roleGet "srmt-admin/internal/http-server/handlers/role/get"
	callbackModsnow "srmt-admin/internal/http-server/handlers/sc/callback/modsnow"
	callbackStock "srmt-admin/internal/http-server/handlers/sc/callback/stock"
	"srmt-admin/internal/http-server/handlers/sc/dc"
	modsnowImg "srmt-admin/internal/http-server/handlers/sc/modsnow/img"
	"srmt-admin/internal/http-server/handlers/sc/modsnow/table"
	"srmt-admin/internal/http-server/handlers/sc/stock"
	"srmt-admin/internal/http-server/handlers/shutdowns"
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
		r.Post("/data/{id}", dataSet.New(deps.Log, deps.PgRepo))
	})

	// Token required routes
	router.Group(func(r chi.Router) {
		r.Use(mwauth.Authenticator(deps.Token))

		r.Get("/auth/me", me.New(deps.Log))

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
			r.Post("/upload/files", upload.New(deps.Log, deps.MinioRepo, deps.PgRepo))

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

			r.Get("/shutdowns", shutdowns.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))
			r.Post("/shutdowns", shutdowns.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/shutdowns/{id}", shutdowns.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/shutdowns/{id}", shutdowns.Delete(deps.Log, deps.PgRepo))

			r.Get("/past-events", pastEventsHandler.Get(deps.Log, deps.PgRepo, deps.MinioRepo, loc))

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
			r.Get("/investments/statuses", investments.GetStatuses(deps.Log, deps.PgRepo))
			r.Get("/investments/{id}", investments.GetByID(deps.Log, deps.PgRepo, deps.MinioRepo))
			r.Post("/investments", investments.Add(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Patch("/investments/{id}", investments.Edit(deps.Log, deps.PgRepo, deps.MinioRepo, deps.PgRepo, deps.PgRepo))
			r.Delete("/investments/{id}", investments.Delete(deps.Log, deps.PgRepo))

			// Invest Active Projects
			r.Get("/invest-active-projects", investActiveProjects.GetAll(deps.Log, deps.PgRepo))
			r.Get("/invest-active-projects/{id}", investActiveProjects.GetByID(deps.Log, deps.PgRepo))
			r.Post("/invest-active-projects", investActiveProjects.Add(deps.Log, deps.PgRepo))
			r.Patch("/invest-active-projects/{id}", investActiveProjects.Edit(deps.Log, deps.PgRepo))
			r.Delete("/invest-active-projects/{id}", investActiveProjects.Delete(deps.Log, deps.PgRepo))
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

			// Receptions
			r.Group(func(r chi.Router) {
				r.Use(mwauth.RequireAnyRole("sc"))

				r.Get("/receptions", receptionGetAll.New(deps.Log, deps.PgRepo, loc))
				r.Get("/receptions/{id}", receptionGetById.New(deps.Log, deps.PgRepo))
				r.Post("/receptions", receptionAdd.New(deps.Log, deps.PgRepo))
				r.Patch("/receptions/{id}", receptionEdit.New(deps.Log, deps.PgRepo))
			})
			r.Delete("/receptions/{id}", receptionDelete.New(deps.Log, deps.PgRepo))

			// Fast Calls
			r.Get("/fast-calls", fastCallGetAll.New(deps.Log, deps.PgRepo))
			r.Get("/fast-calls/{id}", fastCallGetById.New(deps.Log, deps.PgRepo))
			r.Post("/fast-calls", fastCallAdd.New(deps.Log, deps.PgRepo))
			r.Patch("/fast-calls/{id}", fastCallEdit.New(deps.Log, deps.PgRepo))
			r.Delete("/fast-calls/{id}", fastCallDelete.New(deps.Log, deps.PgRepo))
		})
	})
}
