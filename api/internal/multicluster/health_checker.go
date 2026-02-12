package multicluster

import (
	"context"
	"log/slog"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const maxConcurrentHealthChecks = 10

// RunHealthChecker starts a background goroutine that periodically checks the
// health of every cluster in the pool. It updates the in-memory Healthy flag
// and patches the ManagedCluster status on the hub.
func RunHealthChecker(ctx context.Context, pool *ClientPool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run an initial check immediately.
	checkAllClusters(ctx, pool)

	for {
		select {
		case <-ticker.C:
			checkAllClusters(ctx, pool)
		case <-ctx.Done():
			return
		}
	}
}

func checkAllClusters(ctx context.Context, pool *ClientPool) {
	clients := pool.List()
	if len(clients) == 0 {
		return
	}

	// Bounded concurrency: run health checks in parallel with a semaphore.
	sem := make(chan struct{}, maxConcurrentHealthChecks)
	var wg sync.WaitGroup

	for _, cc := range clients {
		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(cc *ClusterClient) {
			defer wg.Done()
			defer func() { <-sem }() // release

			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			healthy := checkCluster(checkCtx, cc)
			cancel()

			cc.SetHealthy(healthy)

			if healthy {
				cc.CircuitBreaker.RecordSuccess()
			} else {
				cc.CircuitBreaker.RecordFailure()
			}

			phase := ClusterPhaseReady
			if !healthy {
				phase = ClusterPhaseUnreachable
			}

			status := map[string]interface{}{
				"phase":             string(phase),
				"kubernetesVersion": cc.K8sVersion,
			}
			if cc.AgentInstalled {
				status["agentInstalled"] = true
			}

			if err := pool.UpdateStatus(ctx, cc.Name, status); err != nil {
				slog.Warn("failed to update cluster status", "cluster", cc.Name, "error", err)
			}
		}(cc)
	}
	wg.Wait()
}

// checkCluster verifies that the K8s API is reachable by calling ServerVersion,
// and discovers the NGF version from the GatewayClass controller name.
func checkCluster(ctx context.Context, cc *ClusterClient) bool {
	if cc.K8sClient == nil {
		return false
	}

	dc := cc.K8sClient.DynamicClient()
	if dc == nil {
		return false
	}

	// Check K8s API reachability by listing a known lightweight resource.
	nsGVR := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	_, err := dc.Resource(nsGVR).List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		slog.Warn("cluster health check failed", "cluster", cc.Name, "error", err)
		return false
	}

	return true
}
