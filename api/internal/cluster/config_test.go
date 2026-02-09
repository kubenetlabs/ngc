package cluster

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		yaml := `clusters:
  - name: "production"
    displayName: "Production US-East"
    kubeconfig: "/etc/ngf/kubeconfigs/prod.yaml"
    default: true
  - name: "staging"
    displayName: "Staging EU-West"
    kubeconfig: "/etc/ngf/kubeconfigs/staging.yaml"
    context: "staging-admin"
`
		path := writeTempFile(t, yaml)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Clusters) != 2 {
			t.Fatalf("expected 2 clusters, got %d", len(cfg.Clusters))
		}
		if cfg.Clusters[0].Name != "production" {
			t.Errorf("expected name production, got %s", cfg.Clusters[0].Name)
		}
		if !cfg.Clusters[0].Default {
			t.Error("expected production to be default")
		}
		if cfg.Clusters[1].Context != "staging-admin" {
			t.Errorf("expected context staging-admin, got %s", cfg.Clusters[1].Context)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/path.yaml")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := writeTempFile(t, "{{invalid yaml")
		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})
}

func TestClustersConfig_Validate(t *testing.T) {
	t.Run("no clusters", func(t *testing.T) {
		cfg := &ClustersConfig{}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for empty clusters")
		}
	})

	t.Run("duplicate names", func(t *testing.T) {
		cfg := &ClustersConfig{
			Clusters: []ClusterEntry{
				{Name: "prod", Kubeconfig: "/a"},
				{Name: "prod", Kubeconfig: "/b"},
			},
		}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for duplicate names")
		}
	})

	t.Run("multiple defaults", func(t *testing.T) {
		cfg := &ClustersConfig{
			Clusters: []ClusterEntry{
				{Name: "prod", Kubeconfig: "/a", Default: true},
				{Name: "staging", Kubeconfig: "/b", Default: true},
			},
		}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for multiple defaults")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		cfg := &ClustersConfig{
			Clusters: []ClusterEntry{
				{Name: "", Kubeconfig: "/a"},
			},
		}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("invalid name format", func(t *testing.T) {
		invalid := []string{"UPPERCASE", "has spaces", "has_underscore", "-starts-with-dash", "ends-with-dash-"}
		for _, name := range invalid {
			cfg := &ClustersConfig{
				Clusters: []ClusterEntry{
					{Name: name, Kubeconfig: "/a"},
				},
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("expected error for invalid name %q", name)
			}
		}
	})

	t.Run("valid names", func(t *testing.T) {
		valid := []string{"a", "prod", "my-cluster", "cluster-1", "a1b2c3"}
		for _, name := range valid {
			cfg := &ClustersConfig{
				Clusters: []ClusterEntry{
					{Name: name, Kubeconfig: "/a"},
				},
			}
			if err := cfg.Validate(); err != nil {
				t.Errorf("unexpected error for valid name %q: %v", name, err)
			}
		}
	})

	t.Run("single default is valid", func(t *testing.T) {
		cfg := &ClustersConfig{
			Clusters: []ClusterEntry{
				{Name: "prod", Kubeconfig: "/a", Default: true},
				{Name: "staging", Kubeconfig: "/b"},
			},
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no default is valid", func(t *testing.T) {
		cfg := &ClustersConfig{
			Clusters: []ClusterEntry{
				{Name: "prod", Kubeconfig: "/a"},
				{Name: "staging", Kubeconfig: "/b"},
			},
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "clusters.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}
