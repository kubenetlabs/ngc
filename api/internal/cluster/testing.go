package cluster

import "github.com/kubenetlabs/ngc/api/internal/kubernetes"

// NewForTest creates a Manager with pre-built clients for testing.
func NewForTest(clients map[string]*kubernetes.Client, defaultName string) *Manager {
	configs := make(map[string]ClusterEntry, len(clients))
	for name := range clients {
		configs[name] = ClusterEntry{
			Name:        name,
			DisplayName: name,
			Default:     name == defaultName,
		}
	}
	return &Manager{
		clients:        clients,
		configs:        configs,
		defaultCluster: defaultName,
	}
}
