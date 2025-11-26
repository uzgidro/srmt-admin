package providers

import (
	"context"
	"log/slog"
	"srmt-admin/internal/config"
	"srmt-admin/internal/storage"
	mongoDriver "srmt-admin/internal/storage/driver/mongo"
	pgDriver "srmt-admin/internal/storage/driver/postgres"
	"srmt-admin/internal/storage/minio"
	mngRepo "srmt-admin/internal/storage/mongo"
	pgRepo "srmt-admin/internal/storage/repo"

	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// StorageProviderSet provides all storage-related dependencies
var StorageProviderSet = wire.NewSet(
	ProvidePostgresDriver,
	ProvidePostgresRepo,
	ProvideMongoClient,
	ProvideMongoRepo,
	ProvideMinioRepo,
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
