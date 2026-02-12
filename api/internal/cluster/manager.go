package cluster

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

// Provider is the interface satisfied by both the file-based Manager and
// the CRD-based PoolAdapter, allowing the API server to swap implementations.
type Provider interface {
	Get(name string) (*kubernetes.Client, error)
	Default() (*kubernetes.Client, error)
	DefaultName() string
	List(ctx context.Context) []ClusterInfo
	Names() []string
}

// ClusterInfo represents a cluster's status as returned to API consumers.
type ClusterInfo struct {
	Name        string             `json:"name"`
	DisplayName string             `json:"displayName"`
	Region      string             `json:"region,omitempty"`
	Environment string             `json:"environment,omitempty"`
	Connected   bool               `json:"connected"`
	Edition     kubernetes.Edition `json:"edition"`
	Default     bool               `json:"default"`
}

// Manager manages multiple Kubernetes cluster clients.
type Manager struct {
	mu             sync.RWMutex
	clients        map[string]*kubernetes.Client
	configs        map[string]ClusterEntry
	defaultCluster string
}

// New creates a Manager from a ClustersConfig, establishing a client for each entry.
func New(cfg *ClustersConfig) (*Manager, error) {
	m := &Manager{
		clients: make(map[string]*kubernetes.Client, len(cfg.Clusters)),
		configs: make(map[string]ClusterEntry, len(cfg.Clusters)),
	}

	for _, entry := range cfg.Clusters {
		client, err := kubernetes.NewFromContext(entry.Kubeconfig, entry.Context)
		if err != nil {
			return nil, fmt.Errorf("creating client for cluster %q: %w", entry.Name, err)
		}
		m.clients[entry.Name] = client
		m.configs[entry.Name] = entry

		if entry.Default {
			m.defaultCluster = entry.Name
		}

		slog.Info("cluster registered", "name", entry.Name, "displayName", entry.DisplayName)
	}

	// If no explicit default, use the first cluster.
	if m.defaultCluster == "" {
		m.defaultCluster = cfg.Clusters[0].Name
	}

	return m, nil
}

// NewSingleCluster wraps a single client as the "default" cluster for backward compatibility.
func NewSingleCluster(client *kubernetes.Client) *Manager {
	return &Manager{
		clients: map[string]*kubernetes.Client{"default": client},
		configs: map[string]ClusterEntry{
			"default": {
				Name:        "default",
				DisplayName: "Default",
				Default:     true,
			},
		},
		defaultCluster: "default",
	}
}

// Get returns the client for the named cluster.
func (m *Manager) Get(name string) (*kubernetes.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.clients[name]
	if !ok {
		return nil, fmt.Errorf("cluster %q not found", name)
	}
	return c, nil
}

// Default returns the default cluster's client.
func (m *Manager) Default() (*kubernetes.Client, error) {
	return m.Get(m.defaultCluster)
}

// DefaultName returns the name of the default cluster.
func (m *Manager) DefaultName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultCluster
}

// List returns information about all registered clusters.
func (m *Manager) List(ctx context.Context) []ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]ClusterInfo, 0, len(m.configs))
	for name, cfg := range m.configs {
		info := ClusterInfo{
			Name:        name,
			DisplayName: cfg.DisplayName,
			Default:     cfg.Default,
		}

		if client, ok := m.clients[name]; ok {
			info.Connected = true
			info.Edition = client.DetectEdition(ctx)
		}

		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// Names returns the names of all registered clusters.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
