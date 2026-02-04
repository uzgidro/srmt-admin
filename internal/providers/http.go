package providers

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/middleware/cors"
	"srmt-admin/internal/http-server/middleware/logger"
	"srmt-admin/internal/http-server/router"
	"srmt-admin/internal/lib/service/alarm"
	"srmt-admin/internal/lib/service/metrics"
	"srmt-admin/internal/lib/service/reservoir"
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
// This replaces the 9 parameters in SetupRoutes
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
) *AppContainer {
	return &AppContainer{
		Router:           r,
		Server:           srv,
		Logger:           log,
		Config:           cfg,
		PgRepo:           pg,
		MongoRepo:        mng,
		MinioRepo:        minioRepo,
		RedisRepo:        redis,
		Token:            tkn,
		Location:         loc,
		MetricsBlender:   metricsBlender,
		ReservoirFetcher: reservoirFetcher,
		HTTPClient:       httpClient,
		AlarmProcessor:   alarmProcessor,
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
		Log:                        log,
		Token:                      tkn,
		PgRepo:                     pg,
		MongoRepo:                  mng,
		MinioRepo:                  minioRepo,
		RedisRepo:                  redis,
		Config:                     *cfg,
		Location:                   loc,
		MetricsBlender:             metricsBlender,
		ReservoirFetcher:           reservoirFetcher,
		HTTPClient:                 httpClient,
		ExcelTemplatePath:          cfg.TemplatePath + "/res-summary.xlsx",
		DischargeExcelTemplatePath: cfg.TemplatePath + "/discharge.xlsx",
		SCExcelTemplatePath:        cfg.TemplatePath + "/sc.xlsx",
		AlarmProcessor:             alarmProcessor,
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
