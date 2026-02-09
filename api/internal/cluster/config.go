package cluster

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// namePattern matches lowercase alphanumeric names with hyphens, 1-63 chars.
var namePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ClusterEntry represents a single cluster's configuration.
type ClusterEntry struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"displayName"`
	Kubeconfig  string `yaml:"kubeconfig"`
	Context     string `yaml:"context"`
	Default     bool   `yaml:"default"`
}

// ClustersConfig is the top-level configuration for multi-cluster support.
type ClustersConfig struct {
	Clusters []ClusterEntry `yaml:"clusters"`
}

// LoadConfig reads and parses a clusters YAML configuration file.
func LoadConfig(path string) (*ClustersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading clusters config: %w", err)
	}

	var cfg ClustersConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing clusters config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating clusters config: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for consistency.
func (c *ClustersConfig) Validate() error {
	if len(c.Clusters) == 0 {
		return fmt.Errorf("no clusters defined")
	}

	names := make(map[string]bool, len(c.Clusters))
	defaultCount := 0

	for i, entry := range c.Clusters {
		if entry.Name == "" {
			return fmt.Errorf("cluster at index %d has empty name", i)
		}
		if !namePattern.MatchString(entry.Name) {
			return fmt.Errorf("cluster name %q is invalid: must be lowercase alphanumeric with hyphens, 1-63 chars", entry.Name)
		}
		if names[entry.Name] {
			return fmt.Errorf("duplicate cluster name %q", entry.Name)
		}
		names[entry.Name] = true

		if entry.Default {
			defaultCount++
		}
	}

	if defaultCount > 1 {
		return fmt.Errorf("multiple clusters marked as default (max 1)")
	}

	return nil
}
