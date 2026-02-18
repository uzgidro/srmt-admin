package providers

import (
	"context"
	"log/slog"
	"srmt-admin/internal/config"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	mongoDriver "srmt-admin/internal/storage/driver/mongo"
	pgDriver "srmt-admin/internal/storage/driver/postgres"
	"srmt-admin/internal/storage/minio"
	mngRepo "srmt-admin/internal/storage/mongo"
	redisRepo "srmt-admin/internal/storage/redis"
	pgRepo "srmt-admin/internal/storage/repo"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// StorageProviderSet provides all storage-related dependencies
var StorageProviderSet = wire.NewSet(
	ProvidePostgresDriver,
	ProvidePostgresRepo,
	ProvideMongoClient,
	ProvideMongoRepo,
	ProvideMinioRepo,
	ProvideRedisClient,
	ProvideRedisRepo,

	// Bindings
	wire.Bind(new(fileupload.FileUploader), new(*minio.Repo)),
)

// ProvidePostgresDriver creates PostgreSQL driver with migrations
func ProvidePostgresDriver(cfg *config.Config, log *slog.Logger) (*storage.Driver, func(), error) {
	driver, err := pgDriver.New(cfg.StoragePath, cfg.MigrationsPath)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		if err := driver.DB.Close(); err != nil {
			log.Error("failed to close postgres connection", "error", err)
		}
	}

	return driver, cleanup, nil
}

// ProvidePostgresRepo creates PostgreSQL repository
func ProvidePostgresRepo(driver *storage.Driver) *pgRepo.Repo {
	return pgRepo.New(driver)
}

// ProvideMongoClient creates MongoDB client
func ProvideMongoClient(cfg config.Mongo, log *slog.Logger) (*mongo.Client, func(), error) {
	client, err := mongoDriver.New(context.Background(), cfg)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Error("failed to disconnect mongo client", "error", err)
		}
	}

	return client, cleanup, nil
}

// ProvideMongoRepo creates MongoDB repository
func ProvideMongoRepo(client *mongo.Client) *mngRepo.Repo {
	return mngRepo.New(client)
}

// ProvideMinioRepo creates MinIO repository
func ProvideMinioRepo(cfg config.Minio, mainCfg *config.Config, log *slog.Logger) (*minio.Repo, error) {
	return minio.New(cfg, log, mainCfg.Bucket)
}

// ProvideRedisClient creates a Redis client with connection pooling
func ProvideRedisClient(cfg config.Redis, log *slog.Logger) (*redis.Client, func(), error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, nil, err
	}

	log.Info("connected to Redis", "addr", cfg.Host+":"+cfg.Port, "db", cfg.DB)

	cleanup := func() {
		if err := client.Close(); err != nil {
			log.Error("failed to close redis connection", "error", err)
		}
	}

	return client, cleanup, nil
}

// ProvideRedisRepo creates the Redis repository for ASUTP telemetry
func ProvideRedisRepo(client *redis.Client, cfg config.ASUTP) *redisRepo.Repo {
	return redisRepo.New(client, cfg.TTL)
}
