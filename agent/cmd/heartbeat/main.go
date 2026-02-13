package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HeartbeatPayload matches the API's HeartbeatRequest.
type HeartbeatPayload struct {
	KubernetesVersion string          `json:"kubernetesVersion"`
	NGFVersion        string          `json:"ngfVersion"`
	ResourceCounts    *ResourceCounts `json:"resourceCounts,omitempty"`
	GPUCapacity       *GPUCapacity    `json:"gpuCapacity,omitempty"`
}

type ResourceCounts struct {
	Gateways        int32 `json:"gateways"`
	HTTPRoutes      int32 `json:"httpRoutes"`
	InferencePools  int32 `json:"inferencePools"`
	InferenceStacks int32 `json:"inferenceStacks"`
	GatewayBundles  int32 `json:"gatewayBundles"`
	Services        int32 `json:"services"`
	Namespaces      int32 `json:"namespaces"`
}

type GPUCapacity struct {
	TotalGPUs     int32            `json:"totalGPUs"`
	AllocatedGPUs int32            `json:"allocatedGPUs"`
	GPUTypes      map[string]int32 `json:"gpuTypes,omitempty"`
}

func main() {
	clusterName := flag.String("cluster-name", os.Getenv("CLUSTER_NAME"), "Name of this cluster")
	hubAPI := flag.String("hub-api", os.Getenv("HUB_API_ENDPOINT"), "Hub API endpoint URL")
	authToken := flag.String("auth-token", os.Getenv("HUB_AUTH_TOKEN"), "Authentication token for hub API")
	interval := flag.Duration("interval", 30*time.Second, "Heartbeat interval")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig (in-cluster if empty)")
	flag.Parse()

	if *clusterName == "" || *hubAPI == "" {
		fmt.Fprintln(os.Stderr, "cluster-name and hub-api are required")
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("starting heartbeat reporter",
		"cluster", *clusterName,
		"hub", *hubAPI,
		"interval", interval.String(),
	)

	// Start health probe server.
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	healthMux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go func() {
		if err := http.ListenAndServe(":8081", healthMux); err != nil {
			slog.Error("health server failed", "error", err)
		}
	}()

	cfg, err := buildConfig(*kubeconfig)
	if err != nil {
		slog.Error("failed to build kubernetes config", "error", err)
		os.Exit(1)
	}

	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		slog.Error("failed to create dynamic client", "error", err)
		os.Exit(1)
	}

	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		slog.Error("failed to create discovery client", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		slog.Info("shutting down")
		cancel()
	}()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	endpoint := fmt.Sprintf("%s/api/v1/clusters/%s/heartbeat", *hubAPI, *clusterName)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	// Send first heartbeat immediately.
	sendHeartbeat(ctx, httpClient, endpoint, *authToken, dc, disco)

	for {
		select {
		case <-ticker.C:
			sendHeartbeat(ctx, httpClient, endpoint, *authToken, dc, disco)
		case <-ctx.Done():
			slog.Info("heartbeat reporter stopped")
			return
		}
	}
}

func sendHeartbeat(ctx context.Context, client *http.Client, endpoint, token string, dc dynamic.Interface, disco *discovery.DiscoveryClient) {
	payload := gatherPayload(ctx, dc, disco)

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal heartbeat", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Error("failed to create request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("heartbeat failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		slog.Warn("heartbeat returned non-success", "status", resp.StatusCode)
		return
	}

	slog.Info("heartbeat sent", "status", resp.StatusCode)
}

func gatherPayload(ctx context.Context, dc dynamic.Interface, disco *discovery.DiscoveryClient) HeartbeatPayload {
	payload := HeartbeatPayload{}

	// K8s version.
	if info, err := disco.ServerVersion(); err == nil {
		payload.KubernetesVersion = info.GitVersion
	}

	// Resource counts.
	counts := &ResourceCounts{}

	gwGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	if list, err := dc.Resource(gwGVR).List(ctx, metav1.ListOptions{}); err == nil {
		counts.Gateways = int32(len(list.Items))
	}

	routeGVR := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	if list, err := dc.Resource(routeGVR).List(ctx, metav1.ListOptions{}); err == nil {
		counts.HTTPRoutes = int32(len(list.Items))
	}

	nsGVR := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	if list, err := dc.Resource(nsGVR).List(ctx, metav1.ListOptions{}); err == nil {
		counts.Namespaces = int32(len(list.Items))
	}

	svcGVR := schema.GroupVersionResource{Version: "v1", Resource: "services"}
	if list, err := dc.Resource(svcGVR).List(ctx, metav1.ListOptions{}); err == nil {
		counts.Services = int32(len(list.Items))
	}

	payload.ResourceCounts = counts

	// GPU capacity from nodes with nvidia.com/gpu.
	nodeGVR := schema.GroupVersionResource{Version: "v1", Resource: "nodes"}
	if list, err := dc.Resource(nodeGVR).List(ctx, metav1.ListOptions{}); err == nil {
		gpu := &GPUCapacity{GPUTypes: make(map[string]int32)}
		for _, node := range list.Items {
			capacity, _, _ := unstructured.NestedMap(node.Object, "status", "capacity")
			if gpuStr, ok := capacity["nvidia.com/gpu"]; ok {
				if gpuVal, ok := gpuStr.(string); ok && gpuVal != "0" {
					count, err := strconv.ParseInt(gpuVal, 10, 32)
					if err != nil || count <= 0 {
						continue
					}
					gpu.TotalGPUs += int32(count)
					labels := node.GetLabels()
					if gpuType, ok := labels["nvidia.com/gpu.product"]; ok {
						gpu.GPUTypes[gpuType] += int32(count)
					}
				}
			}
		}
		if gpu.TotalGPUs > 0 {
			payload.GPUCapacity = gpu
		}
	}

	return payload
}

func buildConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to default kubeconfig.
		home, _ := os.UserHomeDir()
		return clientcmd.BuildConfigFromFlags("", home+"/.kube/config")
	}
	return cfg, nil
}

