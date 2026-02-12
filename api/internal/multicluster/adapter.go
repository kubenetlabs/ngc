package multicluster

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

type editionCacheEntry struct {
	edition   kubernetes.Edition
	expiresAt time.Time
}

const editionCacheTTL = 5 * time.Minute

// PoolAdapter wraps a ClientPool and implements the cluster.Manager interface
// so the API server can swap between file-based and CRD-based cluster management.
type PoolAdapter struct {
	pool           *ClientPool
	defaultCluster string
	editionCache   sync.Map // map[string]editionCacheEntry
}

// NewPoolAdapter creates an adapter that bridges the ClientPool to the
// cluster.Manager interface. The defaultCluster name is used for
// legacy (non-cluster-scoped) API routes.
func NewPoolAdapter(pool *ClientPool, defaultCluster string) *PoolAdapter {
	return &PoolAdapter{
		pool:           pool,
		defaultCluster: defaultCluster,
	}
}

// Pool returns the underlying ClientPool for direct access.
func (a *PoolAdapter) Pool() *ClientPool {
	return a.pool
}

// Get returns the kubernetes.Client for the named cluster.
func (a *PoolAdapter) Get(name string) (*kubernetes.Client, error) {
	cc, err := a.pool.Get(name)
	if err != nil {
		return nil, err
	}
	return cc.K8sClient, nil
}

// Default returns the default cluster's client.
func (a *PoolAdapter) Default() (*kubernetes.Client, error) {
	return a.Get(a.defaultCluster)
}

// DefaultName returns the name of the default cluster.
func (a *PoolAdapter) DefaultName() string {
	return a.defaultCluster
}

// List returns information about all registered clusters.
func (a *PoolAdapter) List(ctx context.Context) []cluster.ClusterInfo {
	clients := a.pool.List()
	infos := make([]cluster.ClusterInfo, 0, len(clients))

	for _, cc := range clients {
		info := cluster.ClusterInfo{
			Name:        cc.Name,
			DisplayName: cc.DisplayName,
			Region:      cc.Region,
			Environment: cc.Environment,
			Connected:   cc.Healthy,
			Default:     cc.Name == a.defaultCluster,
		}
		if cc.K8sClient != nil {
			info.Edition = a.cachedEdition(ctx, cc)
		}
		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// Names returns the names of all registered clusters.
func (a *PoolAdapter) Names() []string {
	names := a.pool.Names()
	sort.Strings(names)
	return names
}

// cachedEdition returns the cached edition for a cluster, refreshing if expired.
func (a *PoolAdapter) cachedEdition(ctx context.Context, cc *ClusterClient) kubernetes.Edition {
	if entry, ok := a.editionCache.Load(cc.Name); ok {
		cached := entry.(editionCacheEntry)
		if time.Now().Before(cached.expiresAt) {
			return cached.edition
		}
	}
	edition := cc.K8sClient.DetectEdition(ctx)
	a.editionCache.Store(cc.Name, editionCacheEntry{
		edition:   edition,
		expiresAt: time.Now().Add(editionCacheTTL),
	})
	return edition
}
