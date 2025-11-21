package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
)

// ReservoirConfig represents the static reservoir API configuration
type ReservoirConfig struct {
	BaseURL string            `yaml:"base_url" validate:"required,url"`
	Sources []ReservoirSource `yaml:"sources"`
}

// ReservoirSource represents a single reservoir data source
type ReservoirSource struct {
	APIID          int   `yaml:"api_id" validate:"required"`
	OrganizationID int64 `yaml:"organization_id" validate:"required"`
}

// LoadReservoirConfig loads the reservoir configuration from the specified file path
func LoadReservoirConfig(path string) (*ReservoirConfig, error) {
	var cfg ReservoirConfig
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read reservoir config: %w", err)
	}
	return &cfg, nil
}
