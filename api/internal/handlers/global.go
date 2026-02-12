package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	mc "github.com/kubenetlabs/ngc/api/internal/multicluster"
)

const globalQueryTimeout = 10 * time.Second

// GlobalHandler serves cross-cluster aggregation endpoints.
type GlobalHandler struct {
	Pool    *mc.ClientPool
	Manager cluster.Provider
}

// clusterGateway wraps a gateway response with its source cluster.
type clusterGateway struct {
	ClusterName   string          `json:"clusterName"`
	ClusterRegion string          `json:"clusterRegion"`
	Gateway       GatewayResponse `json:"gateway"`
}

// clusterRoute wraps a route response with its source cluster.
type clusterRoute struct {
	ClusterName   string `json:"clusterName"`
	ClusterRegion string `json:"clusterRegion"`
	Route         any    `json:"route"`
}

// gpuClusterCapacity represents GPU capacity for a single cluster.
type gpuClusterCapacity struct {
	ClusterName   string         `json:"clusterName"`
	ClusterRegion string         `json:"clusterRegion"`
	TotalGPUs     int            `json:"totalGPUs"`
	AllocatedGPUs int            `json:"allocatedGPUs"`
	GPUTypes      map[string]int `json:"gpuTypes,omitempty"`
}

// globalGPUCapacity is the aggregated GPU capacity across all clusters.
type globalGPUCapacity struct {
	TotalGPUs     int                  `json:"totalGPUs"`
	AllocatedGPUs int                  `json:"allocatedGPUs"`
	Clusters      []gpuClusterCapacity `json:"clusters"`
}

// Gateways lists gateways from all clusters in parallel.
func (h *GlobalHandler) Gateways(w http.ResponseWriter, r *http.Request) {
	clusters := h.Manager.List(r.Context())
	ns := r.URL.Query().Get("namespace")

	type result struct {
		clusterName   string
		clusterRegion string
		gateways      []GatewayResponse
		err           error
	}

	results := make([]result, len(clusters))
	var wg sync.WaitGroup

	for i, ci := range clusters {
		wg.Add(1)
		go func(idx int, info cluster.ClusterInfo) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), globalQueryTimeout)
			defer cancel()
			k8s, err := h.Manager.Get(info.Name)
			if err != nil {
				results[idx] = result{clusterName: info.Name, err: err}
				return
			}
			gateways, err := k8s.ListGateways(ctx, ns)
			if err != nil {
				results[idx] = result{clusterName: info.Name, err: err}
				return
			}
			resp := make([]GatewayResponse, 0, len(gateways))
			for j := range gateways {
				resp = append(resp, toGatewayResponse(&gateways[j]))
			}
			results[idx] = result{clusterName: info.Name, clusterRegion: info.Region, gateways: resp}
		}(i, ci)
	}
	wg.Wait()

	var allGateways []clusterGateway
	for _, res := range results {
		if res.err != nil {
			continue
		}
		for _, gw := range res.gateways {
			allGateways = append(allGateways, clusterGateway{
				ClusterName:   res.clusterName,
				ClusterRegion: res.clusterRegion,
				Gateway:       gw,
			})
		}
	}
	if allGateways == nil {
		allGateways = []clusterGateway{}
	}
	writeJSON(w, http.StatusOK, allGateways)
}

// Routes lists HTTP routes from all clusters in parallel.
func (h *GlobalHandler) Routes(w http.ResponseWriter, r *http.Request) {
	clusters := h.Manager.List(r.Context())

	type result struct {
		clusterName   string
		clusterRegion string
		routes        []HTTPRouteResponse
		err           error
	}

	results := make([]result, len(clusters))
	var wg sync.WaitGroup

	for i, ci := range clusters {
		wg.Add(1)
		go func(idx int, info cluster.ClusterInfo) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(r.Context(), globalQueryTimeout)
			defer cancel()
			k8s, err := h.Manager.Get(info.Name)
			if err != nil {
				results[idx] = result{clusterName: info.Name, err: err}
				return
			}
			routes, err := k8s.ListHTTPRoutes(ctx, "")
			if err != nil {
				results[idx] = result{clusterName: info.Name, err: err}
				return
			}
			resp := make([]HTTPRouteResponse, 0, len(routes))
			for j := range routes {
				resp = append(resp, toHTTPRouteResponse(&routes[j]))
			}
			results[idx] = result{clusterName: info.Name, clusterRegion: info.Region, routes: resp}
		}(i, ci)
	}
	wg.Wait()

	var allRoutes []clusterRoute
	for _, res := range results {
		if res.err != nil {
			continue
		}
		for _, rt := range res.routes {
			allRoutes = append(allRoutes, clusterRoute{
				ClusterName:   res.clusterName,
				ClusterRegion: res.clusterRegion,
				Route:         rt,
			})
		}
	}
	if allRoutes == nil {
		allRoutes = []clusterRoute{}
	}
	writeJSON(w, http.StatusOK, allRoutes)
}

// GPUCapacity aggregates GPU capacity across all clusters.
func (h *GlobalHandler) GPUCapacity(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		writeJSON(w, http.StatusOK, globalGPUCapacity{Clusters: []gpuClusterCapacity{}})
		return
	}

	clusterClients := h.Pool.List()
	var totalGPUs, allocatedGPUs int
	clusterCapacities := make([]gpuClusterCapacity, 0, len(clusterClients))

	for _, cc := range clusterClients {
		cap := queryGPUCapacity(r.Context(), cc)
		totalGPUs += cap.TotalGPUs
		allocatedGPUs += cap.AllocatedGPUs
		clusterCapacities = append(clusterCapacities, cap)
	}

	writeJSON(w, http.StatusOK, globalGPUCapacity{
		TotalGPUs:     totalGPUs,
		AllocatedGPUs: allocatedGPUs,
		Clusters:      clusterCapacities,
	})
}

// queryGPUCapacity queries a single cluster for GPU capacity.
func queryGPUCapacity(_ context.Context, cc *mc.ClusterClient) gpuClusterCapacity {
	return gpuClusterCapacity{
		ClusterName:   cc.Name,
		ClusterRegion: cc.Region,
		TotalGPUs:     0,
		AllocatedGPUs: 0,
	}
}
