package providers

import (
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/lib/service/ascue"
	"srmt-admin/internal/lib/service/reservoir"
	"srmt-admin/internal/token"
	"time"

	"github.com/google/wire"
)

// ServiceProviderSet provides all business service dependencies
var ServiceProviderSet = wire.NewSet(
	ProvideTokenService,
	ProvideASCUEFetcher,
	ProvideReservoirFetcher,
	ProvideHTTPClient,
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
