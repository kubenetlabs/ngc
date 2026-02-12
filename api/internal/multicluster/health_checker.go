package multicluster

import (
	"context"
	"log/slog"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
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

			cc.mu.RLock()
			status := map[string]interface{}{
				"phase":             string(phase),
				"kubernetesVersion": cc.K8sVersion,
				"ngfVersion":        cc.NGFVersion,
				"agentInstalled":    cc.AgentInstalled,
			}
			if cc.ResourceCounts != nil {
				status["resourceCounts"] = map[string]interface{}{
					"gateways":   cc.ResourceCounts.Gateways,
					"httpRoutes": cc.ResourceCounts.HTTPRoutes,
				}
			}
			cc.mu.RUnlock()

			if err := pool.UpdateStatus(ctx, cc.Name, status); err != nil {
				slog.Warn("failed to update cluster status", "cluster", cc.Name, "error", err)
			}
		}(cc)
	}
	wg.Wait()
}

// checkCluster verifies that the K8s API is reachable, discovers the K8s version,
// NGF version from GatewayClass, and counts Gateway API resources.
func checkCluster(ctx context.Context, cc *ClusterClient) bool {
	if cc.K8sClient == nil {
		return false
	}

	dc := cc.K8sClient.DynamicClient()
	if dc == nil {
		return false
	}

	// Discover K8s version via the discovery client.
	k8sVersion := discoverK8sVersion(ctx, cc)

	// Count Gateway API resources.
	gateways := countResources(ctx, dc, schema.GroupVersionResource{
		Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways",
	})
	httpRoutes := countResources(ctx, dc, schema.GroupVersionResource{
		Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes",
	})

	// Discover NGF version from GatewayClass controller name.
	ngfVersion := discoverNGFVersion(ctx, dc)

	// Update the ClusterClient fields.
	cc.mu.Lock()
	if k8sVersion != "" {
		cc.K8sVersion = k8sVersion
	}
	if ngfVersion != "" {
		cc.NGFVersion = ngfVersion
	}
	cc.ResourceCounts = &ResourceCounts{
		Gateways:   gateways,
		HTTPRoutes: httpRoutes,
	}
	cc.mu.Unlock()

	return true
}

// discoverK8sVersion uses the discovery client to get the server version.
func discoverK8sVersion(ctx context.Context, cc *ClusterClient) string {
	restConfig := cc.K8sClient.RestConfig()
	if restConfig == nil {
		return ""
	}
	dc, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		slog.Debug("failed to create discovery client", "cluster", cc.Name, "error", err)
		return ""
	}
	info, err := dc.ServerVersion()
	if err != nil {
		slog.Warn("failed to get server version", "cluster", cc.Name, "error", err)
		return ""
	}
	return info.GitVersion
}

// countResources counts all instances of a given GVR across all namespaces.
func countResources(ctx context.Context, dc dynamic.Interface, gvr schema.GroupVersionResource) int32 {
	list, err := dc.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0
	}
	return int32(len(list.Items))
}

// discoverNGFVersion finds the NGF version from the GatewayClass description or
// controller name. Looks for a GatewayClass with controller containing "nginx".
func discoverNGFVersion(ctx context.Context, dc dynamic.Interface) string {
	gcGVR := schema.GroupVersionResource{
		Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gatewayclasses",
	}
	list, err := dc.Resource(gcGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return ""
	}
	for _, item := range list.Items {
		spec, _ := item.Object["spec"].(map[string]interface{})
		controllerName, _ := spec["controllerName"].(string)
		if controllerName == "gateway.nginx.org/nginx-gateway-controller" {
			// Try to get version from the description or labels.
			annotations := item.GetAnnotations()
			if v, ok := annotations["nginx-gateway-fabric/version"]; ok {
				return v
			}
			labels := item.GetLabels()
			if v, ok := labels["app.kubernetes.io/version"]; ok {
				return v
			}
			// Return the controller name as a fallback indicator that NGF exists.
			return "installed"
		}
	}
	return ""
}
