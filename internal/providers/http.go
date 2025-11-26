package providers

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/http-server/middleware/cors"
	"srmt-admin/internal/http-server/middleware/logger"
	"srmt-admin/internal/http-server/router"
	"srmt-admin/internal/lib/service/ascue"
	"srmt-admin/internal/lib/service/reservoir"
	"srmt-admin/internal/storage/minio"
	mngRepo "srmt-admin/internal/storage/mongo"
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
	Token            *token.Token
	Location         *time.Location
	ASCUEFetcher     *ascue.Fetcher
	ReservoirFetcher *reservoir.Fetcher
	HTTPClient       *http.Client
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
	tkn *token.Token,
	loc *time.Location,
	ascueFetcher *ascue.Fetcher,
	reservoirFetcher *reservoir.Fetcher,
	httpClient *http.Client,
) *AppContainer {
	return &AppContainer{
		Router:           r,
		Server:           srv,
		Logger:           log,
		Config:           cfg,
		PgRepo:           pg,
		MongoRepo:        mng,
		MinioRepo:        minioRepo,
		Token:            tkn,
		Location:         loc,
		ASCUEFetcher:     ascueFetcher,
		ReservoirFetcher: reservoirFetcher,
		HTTPClient:       httpClient,
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
	loc *time.Location,
	ascueFetcher *ascue.Fetcher,
	reservoirFetcher *reservoir.Fetcher,
	httpClient *http.Client,
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
		Log:              log,
		Token:            tkn,
		PgRepo:           pg,
		MongoRepo:        mng,
		MinioRepo:        minioRepo,
		Config:           *cfg,
		Location:         loc,
		ASCUEFetcher:     ascueFetcher,
		ReservoirFetcher: reservoirFetcher,
		HTTPClient:       httpClient,
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
