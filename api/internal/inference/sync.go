package inference

import (
	"context"
	"log/slog"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	mc "github.com/kubenetlabs/ngc/api/internal/multicluster"
)

var inferenceStackGVR = schema.GroupVersionResource{
	Group:    "ngf-console.f5.com",
	Version:  "v1alpha1",
	Resource: "inferencestacks",
}

// RunSyncLoop periodically lists InferenceStack CRDs across all clusters
// and upserts their current state into ClickHouse via the MetricsProvider.
func RunSyncLoop(ctx context.Context, pool *mc.ClientPool, provider MetricsProvider, interval time.Duration) {
	slog.Info("inference pool sync loop starting", "interval", interval)

	syncAllClusters(ctx, pool, provider)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			syncAllClusters(ctx, pool, provider)
		case <-ctx.Done():
			slog.Info("inference pool sync loop stopped")
			return
		}
	}
}

func syncAllClusters(ctx context.Context, pool *mc.ClientPool, provider MetricsProvider) {
	clients := pool.List()
	if len(clients) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, cc := range clients {
		wg.Add(1)
		go func(cc *mc.ClusterClient) {
			defer wg.Done()
			syncCluster(ctx, cc, provider)
		}(cc)
	}
	wg.Wait()
}

func syncCluster(ctx context.Context, cc *mc.ClusterClient, provider MetricsProvider) {
	if cc.K8sClient == nil {
		return
	}
	dc := cc.K8sClient.DynamicClient()
	if dc == nil {
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	list, err := dc.Resource(inferenceStackGVR).Namespace("").List(listCtx, metav1.ListOptions{})
	if err != nil {
		slog.Warn("sync: failed to list InferenceStacks", "cluster", cc.Name, "error", err)
		return
	}

	clusterCtx := cluster.WithClusterName(ctx, cc.Name)

	for i := range list.Items {
		ps := crdToPoolStatus(&list.Items[i])
		if err := provider.UpsertPool(clusterCtx, ps); err != nil {
			slog.Warn("sync: failed to upsert pool", "cluster", cc.Name, "pool", ps.Name, "error", err)
		}
	}

	slog.Debug("sync: completed", "cluster", cc.Name, "pools", len(list.Items))
}

// crdToPoolStatus extracts PoolStatus fields from an unstructured InferenceStack.
func crdToPoolStatus(obj *unstructured.Unstructured) PoolStatus {
	ps := PoolStatus{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		CreatedAt: obj.GetCreationTimestamp().Time,
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec != nil {
		ps.ModelName, _, _ = unstructured.NestedString(spec, "modelName")
		ps.ModelVersion, _, _ = unstructured.NestedString(spec, "modelVersion")
		ps.ServingBackend, _, _ = unstructured.NestedString(spec, "servingBackend")

		pool, _, _ := unstructured.NestedMap(spec, "pool")
		if pool != nil {
			ps.GPUType, _, _ = unstructured.NestedString(pool, "gpuType")
			gpuCount, _, _ := unstructured.NestedInt64(pool, "gpuCount")
			ps.GPUCount = uint32(gpuCount)
			replicas, _, _ := unstructured.NestedInt64(pool, "replicas")
			ps.Replicas = uint32(replicas)
			minReplicas, _, _ := unstructured.NestedInt64(pool, "minReplicas")
			ps.MinReplicas = uint32(minReplicas)
			maxReplicas, _, _ := unstructured.NestedInt64(pool, "maxReplicas")
			ps.MaxReplicas = uint32(maxReplicas)
		}
	}

	// Extract status phase from the CRD; default to "Pending" if not set.
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status != nil {
		phase, _, _ := unstructured.NestedString(status, "phase")
		if phase != "" {
			ps.Status = phase
		} else {
			ps.Status = "Pending"
		}
	} else {
		ps.Status = "Pending"
	}

	return ps
}
