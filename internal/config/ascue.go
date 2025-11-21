package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
)

// ASCUEConfig represents the ASCUE (automated system for commercial electricity accounting) configuration
type ASCUEConfig struct {
	Sources []Source `yaml:"sources"`
}

// Source represents a single ASCUE data source (typically a cascade)
type Source struct {
	URL            string           `yaml:"url" validate:"required,url"`
	OrganizationID int64            `yaml:"organization_id" validate:"required"`
	Metrics        MetricMapping    `yaml:"metrics"`
	Aggregates     AggregateMapping `yaml:"aggregates"`
	Children       []ChildOrg       `yaml:"children,omitempty"`
}

// MetricMapping maps metric names to their array index in the API response
type MetricMapping struct {
	Active      int  `yaml:"active"`
	Reactive    int  `yaml:"reactive"`
	PowerImport *int `yaml:"power_import,omitempty"`
	PowerExport *int `yaml:"power_export,omitempty"`
	OwnNeeds    *int `yaml:"own_needs,omitempty"`
	Flow        *int `yaml:"flow,omitempty"`
}

// AggregateMapping maps aggregate counters to their array index in the API response
type AggregateMapping struct {
	Active  int `yaml:"active"`
	Pending int `yaml:"pending"`
	Repair  int `yaml:"repair"`
}

// ChildOrg represents a child organization (GES/mini/micro) under a cascade
type ChildOrg struct {
	OrganizationID int64            `yaml:"organization_id" validate:"required"`
	Metrics        MetricMapping    `yaml:"metrics"`
	Aggregates     AggregateMapping `yaml:"aggregates"`
}

// LoadASCUEConfig loads the ASCUE configuration from the specified file path
func LoadASCUEConfig(path string) (*ASCUEConfig, error) {
	var cfg ASCUEConfig
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read ASCUE config: %w", err)
	}
	return &cfg, nil
}
