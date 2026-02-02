package providers

import (
	"log/slog"
	"os"
	"srmt-admin/internal/config"
	"time"

	"github.com/google/wire"
)

// ConfigProviderSet provides all configuration-related dependencies
var ConfigProviderSet = wire.NewSet(
	ProvideConfig,
	ProvideLogger,
	ProvideJwtConfig,
	ProvideMongoConfig,
	ProvideMinioConfig,
	ProvideLocation,
	ProvideASCUEConfig,
	ProvideReservoirConfig,
	ProvideRedisConfig,
	ProvideASUTPConfig,
)

// ProvideConfig loads the main application config
func ProvideConfig() *config.Config {
	return config.MustLoad()
}

// ProvideLogger creates a logger based on environment
func ProvideLogger(cfg *config.Config) *slog.Logger {
	switch cfg.Env {
	case "local", "dev":
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "prod":
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
}

// ProvideJwtConfig extracts JWT config from main config
func ProvideJwtConfig(cfg *config.Config) config.JwtConfig {
	return cfg.JwtConfig
}

// ProvideMongoConfig extracts Mongo config from main config
func ProvideMongoConfig(cfg *config.Config) config.Mongo {
	return cfg.Mongo
}

// ProvideMinioConfig extracts MinIO config from main config
func ProvideMinioConfig(cfg *config.Config) config.Minio {
	return cfg.Minio
}

// ProvideLocation extracts timezone location
func ProvideLocation(cfg *config.Config) *time.Location {
	return cfg.GetLocation()
}

// ProvideASCUEConfig loads ASCUE config (returns nil on error)
func ProvideASCUEConfig(log *slog.Logger) *config.ASCUEConfig {
	cfg, err := config.LoadASCUEConfig("config/ascue.yaml")
	if err != nil {
		log.Warn("failed to load ASCUE config, cascades will not include ASCUE metrics", "error", err)
		return nil
	}
	return cfg
}

// ProvideReservoirConfig loads reservoir config (returns nil on error)
func ProvideReservoirConfig(log *slog.Logger) *config.ReservoirConfig {
	cfg, err := config.LoadReservoirConfig("config/reservoir.yaml")
	if err != nil {
		log.Warn("failed to load reservoir config, organizations will not include reservoir metrics", "error", err)
		return nil
	}
	return cfg
}

// ProvideRedisConfig extracts Redis config from main config
func ProvideRedisConfig(cfg *config.Config) config.Redis {
	return cfg.Redis
}

// ProvideASUTPConfig extracts ASUTP config from main config
func ProvideASUTPConfig(cfg *config.Config) config.ASUTP {
	return cfg.ASUTP
}
