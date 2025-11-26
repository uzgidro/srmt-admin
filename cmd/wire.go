//go:build wireinject
// +build wireinject

package main

import (
	"srmt-admin/internal/providers"

	"github.com/google/wire"
)

// InitializeApp builds the application with all dependencies
func InitializeApp() (*providers.AppContainer, func(), error) {
	wire.Build(
		providers.ConfigProviderSet,
		providers.StorageProviderSet,
		providers.ServiceProviderSet,
		providers.HTTPProviderSet,
	)
	return nil, nil, nil
}
